package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/strangelove-ventures/horcrux-proxy/signer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	namespaceFile     = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	labelCosmosSentry = "app.kubernetes.io/component=cosmos-sentry"
)

func watchForChangedSentries(
	ctx context.Context,
	a *appState,
	all bool, // should we connect to sentries on all nodes, or just this node?
) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get in cluster config: %w", err)
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kube clientset: %w", err)
	}

	thisNode := ""
	if !all {
		// need to determine which node this pod is on so we can only connect to sentries on this node

		nsbz, err := os.ReadFile(namespaceFile)
		if err != nil {
			return fmt.Errorf("failed to read namespace from service account: %w", err)
		}
		ns := string(nsbz)

		thisPod, err := clientset.CoreV1().Pods(ns).Get(ctx, os.Getenv("HOSTNAME"), metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get this pod: %w", err)
		}

		thisNode = thisPod.Spec.NodeName
	}

	t := time.NewTimer(30 * time.Second)

	go func() {
		defer t.Stop()
		for {
			if err := reconcileSentries(ctx, a, thisNode, clientset, all); err != nil {
				a.logger.Error("Failed to reconcile sentries with kube api", "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				t.Reset(30 * time.Second)
			}
		}
	}()

	return nil
}

func reconcileSentries(
	ctx context.Context,
	a *appState,
	thisNode string,
	clientset *kubernetes.Clientset,
	all bool, // should we connect to sentries on all nodes, or just this node?
) error {
	configNodes := make([]string, 0)

	services, err := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{
		LabelSelector: labelCosmosSentry,
	})

	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, s := range services.Items {
		if len(s.Spec.Ports) != 1 || s.Spec.Ports[0].Name != "sentry-privval" {
			continue
		}

		set := labels.Set(s.Spec.Selector)

		pods, err := clientset.CoreV1().Pods(s.Namespace).List(ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()})
		if err != nil {
			return fmt.Errorf("failed to list pods in namespace %s for service %s: %w", s.Namespace, s.Name, err)
		}

		if len(pods.Items) != 1 {
			continue
		}

		if !all && pods.Items[0].Spec.NodeName != thisNode {
			continue
		}

		// Connect to this service
		configNodes = append(configNodes, fmt.Sprintf("tcp://%s.%s:%d", s.Name, s.Namespace, s.Spec.Ports[0].Port))
	}

	newSentries := make([]string, 0)

	for _, newConfigSentry := range configNodes {
		foundNewConfigSentry := false
		for existingSentry := range a.sentries {
			if existingSentry == newConfigSentry {
				foundNewConfigSentry = true
				break
			}
		}
		if !foundNewConfigSentry {
			a.logger.Info("Will add new sentry", "address", newConfigSentry)
			newSentries = append(newSentries, newConfigSentry)
		}
	}

	removedSentries := make([]string, 0)

	for existingSentry := range a.sentries {
		foundExistingSentry := false
		for _, newConfigSentry := range configNodes {
			if existingSentry == newConfigSentry {
				foundExistingSentry = true
				break
			}
		}
		if !foundExistingSentry {
			a.logger.Info("Will remove existing sentry", "address", existingSentry)
			removedSentries = append(removedSentries, existingSentry)
		}
	}

	for _, s := range removedSentries {
		if err := a.sentries[s].Stop(); err != nil {
			return fmt.Errorf("failed to stop remote signer: %w", err)
		}
		delete(a.sentries, s)
	}

	for _, newSentry := range newSentries {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		s := signer.NewReconnRemoteSigner(newSentry, a.logger, a.loadBalancer, dialer)

		if err := s.Start(); err != nil {
			return fmt.Errorf("failed to start new remote signer(s): %w", err)
		}
		a.sentries[newSentry] = s
	}

	return nil
}
