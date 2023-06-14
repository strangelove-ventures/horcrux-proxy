package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"

	"github.com/strangelove-ventures/horcrux-proxy/config"
	"github.com/strangelove-ventures/horcrux-proxy/signer"
)

func watchForConfigFileUpdates(
	ctx context.Context,
	a *appState,
) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Errorf("watcher setup failed: %w", err))
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		defer close(done)

		var mu sync.Mutex

		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}
				mu.Lock()
				if err := configUpdate(a); err != nil {
					a.logger.Error("Error during config reconcile", "err", err)
				}
				mu.Unlock()
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				a.logger.Error("Watcher error", "err", err)
			}
		}
	}()

	if err := watcher.Add(a.config.ConfigFile); err != nil {
		panic(fmt.Errorf("add watcher failed: %w", err))
	}
	<-done
}

func configUpdate(
	a *appState,
) error {
	bz, err := os.ReadFile(a.config.ConfigFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}
	var config config.Config
	if err := yaml.Unmarshal(bz, &config); err != nil {
		return fmt.Errorf("error unmarshalling config file: %w", err)
	}

	newSentries := make([]string, 0)
	removedSentries := make([]string, 0)

	configNodes := config.Nodes()

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
		s := signer.NewReconnRemoteSigner(newSentry, a.logger, a.listener, dialer)

		if err := s.Start(); err != nil {
			return fmt.Errorf("failed to start new remote signer(s): %w", err)
		}
		a.sentries[newSentry] = s
	}

	a.config.Config = config

	return nil
}
