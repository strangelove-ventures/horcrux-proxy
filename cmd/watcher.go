package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	cometlog "github.com/cometbft/cometbft/libs/log"
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

type SentryWatcher struct {
	all                bool
	client             *kubernetes.Clientset
	hc                 signer.HorcruxConnection
	log                cometlog.Logger
	node               string
	operator           bool
	persistentSentries []*signer.ReconnRemoteSigner
	sentries           map[string]*signer.ReconnRemoteSigner

	stop chan struct{}
	done chan struct{}
}

func NewSentryWatcher(
	ctx context.Context,
	logger cometlog.Logger,
	all bool, // should we connect to sentries on all nodes, or just this node?
	hc signer.HorcruxConnection,
	operator bool,
	sentries []string,
	maxReadSize int,
) (*SentryWatcher, error) {
	var clientset *kubernetes.Clientset
	var thisNode string

	if operator {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get in cluster config: %w", err)
		}
		// creates the clientset
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create kube clientset: %w", err)
		}

		if !all {
			// need to determine which node this pod is on so we can only connect to sentries on this node

			nsbz, err := os.ReadFile(namespaceFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read namespace from service account: %w", err)
			}
			ns := string(nsbz)

			thisPod, err := clientset.CoreV1().Pods(ns).Get(ctx, os.Getenv("HOSTNAME"), metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get this pod: %w", err)
			}

			thisNode = thisPod.Spec.NodeName
		}
	}

	persistentSentries := make([]*signer.ReconnRemoteSigner, len(sentries))
	for i, sentry := range sentries {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		persistentSentries[i] = signer.NewReconnRemoteSigner(sentry, logger, hc, dialer, maxReadSize)
	}

	return &SentryWatcher{
		all:                all,
		client:             clientset,
		done:               make(chan struct{}),
		hc:                 hc,
		log:                logger,
		node:               thisNode,
		operator:           operator,
		persistentSentries: persistentSentries,
		sentries:           make(map[string]*signer.ReconnRemoteSigner),
		stop:               make(chan struct{}),
	}, nil
}

// Watch will reconcile the sentries with the kube api at a reasonable interval.
// It must be called only once.
func (w *SentryWatcher) Watch(ctx context.Context, maxReadSize int) {
	for _, sentry := range w.persistentSentries {
		if err := sentry.Start(); err != nil {
			w.log.Error("Failed to start persistent sentry", "error", err)
		}
	}
	if !w.operator {
		return
	}
	defer close(w.done)
	const interval = 30 * time.Second
	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		if err := w.reconcileSentries(ctx, maxReadSize); err != nil {
			w.log.Error("Failed to reconcile sentries with kube api", "error", err)
		}
		select {
		case <-w.stop:
			return
		case <-ctx.Done():
			return
		case <-timer.C:
			timer.Reset(interval)
		}
	}
}

// Stop cleans up the sentries and stops the watcher. It must be called only once.
func (w *SentryWatcher) Stop() error {
	// The dual channel synchronization ensures w.sentries is only read/mutated by one goroutine.
	close(w.stop)
	<-w.done
	var err error
	for _, sentry := range w.persistentSentries {
		err = errors.Join(err, sentry.Stop())
	}
	for _, sentry := range w.sentries {
		err = errors.Join(err, sentry.Stop())
	}
	return err
}

func (w *SentryWatcher) reconcileSentries(
	ctx context.Context,
	maxReadSize int,
) error {
	configNodes := make([]string, 0)

	services, err := w.client.CoreV1().Services("").List(ctx, metav1.ListOptions{
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

		pods, err := w.client.CoreV1().Pods(s.Namespace).List(ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()})
		if err != nil {
			return fmt.Errorf("failed to list pods in namespace %s for service %s: %w", s.Namespace, s.Name, err)
		}

		if len(pods.Items) != 1 {
			continue
		}

		if !w.all && pods.Items[0].Spec.NodeName != w.node {
			continue
		}

		// Connect to this service
		configNodes = append(configNodes, fmt.Sprintf("tcp://%s.%s:%d", s.Name, s.Namespace, s.Spec.Ports[0].Port))
	}

	newSentries := make([]string, 0)

	for _, newConfigSentry := range configNodes {
		foundNewConfigSentry := false
		for existingSentry := range w.sentries {
			if existingSentry == newConfigSentry {
				foundNewConfigSentry = true
				break
			}
		}
		if !foundNewConfigSentry {
			w.log.Info("Will add new sentry", "address", newConfigSentry)
			newSentries = append(newSentries, newConfigSentry)
		}
	}

	removedSentries := make([]string, 0)

	for existingSentry := range w.sentries {
		foundExistingSentry := false
		for _, newConfigSentry := range configNodes {
			if existingSentry == newConfigSentry {
				foundExistingSentry = true
				break
			}
		}
		if !foundExistingSentry {
			w.log.Info("Will remove existing sentry", "address", existingSentry)
			removedSentries = append(removedSentries, existingSentry)
		}
	}

	for _, s := range removedSentries {
		if err := w.sentries[s].Stop(); err != nil {
			return fmt.Errorf("failed to stop remote signer: %w", err)
		}
		delete(w.sentries, s)
	}

	for _, newSentry := range newSentries {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		s := signer.NewReconnRemoteSigner(newSentry, w.log, w.hc, dialer, maxReadSize)

		if err := s.Start(); err != nil {
			return fmt.Errorf("failed to start new remote signer(s): %w", err)
		}
		w.sentries[newSentry] = s
	}

	return nil
}
