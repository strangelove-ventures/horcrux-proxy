package cmd

import (
	"fmt"

	cometlog "github.com/cometbft/cometbft/libs/log"
	cometos "github.com/cometbft/cometbft/libs/os"
	"github.com/spf13/cobra"

	"github.com/strangelove-ventures/horcrux-proxy/privval"
	"github.com/strangelove-ventures/horcrux-proxy/signer"
)

const (
	flagLogLevel = "log-level"
	flagListen   = "listen"
	flagAll      = "all"
)

func startCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "start",
		Short:        "Start horcrux-proxy process",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			logLevel, _ := cmd.Flags().GetString(flagLogLevel)
			logLevelOpt, err := cometlog.AllowLevel(logLevel)
			if err != nil {
				return fmt.Errorf("failed to parse log level: %w", err)
			}

			a.logger = cometlog.NewFilter(cometlog.NewTMLogger(cometlog.NewSyncWriter(out)), logLevelOpt).With("module", "validator")

			a.logger.Info("Horcrux Proxy")

			listenAddrs, _ := cmd.Flags().GetStringArray(flagListen)
			all, _ := cmd.Flags().GetBool(flagAll)

			listeners := make([]privval.SignerListener, len(listenAddrs))
			for i, addr := range listenAddrs {
				listeners[i] = privval.NewSignerListener(a.logger, addr)
			}

			a.loadBalancer = privval.NewRemoteSignerLoadBalancer(a.logger, listeners)

			if err := a.loadBalancer.Start(); err != nil {
				return fmt.Errorf("failed to start listener(s): %w", err)
			}

			a.sentries = make(map[string]*signer.ReconnRemoteSigner)

			if err := watchForChangedSentries(cmd.Context(), a, all); err != nil {
				return err
			}

			waitAndTerminate(a)

			return nil
		},
	}

	cmd.Flags().StringArrayP(flagListen, "l", []string{"tcp://0.0.0.0:1234"}, "Privval listen addresses for the proxy")
	cmd.Flags().BoolP(flagAll, "a", false, "Connect to sentries on all nodes")
	cmd.Flags().String(flagLogLevel, "info", "Set log level (debug, info, error, none)")

	return cmd
}

func waitAndTerminate(a *appState) {
	done := make(chan struct{})
	cometos.TrapSignal(a.logger, func() {
		for _, s := range a.sentries {
			err := s.Stop()
			if err != nil {
				panic(err)
			}
		}
		if err := a.loadBalancer.Stop(); err != nil {
			panic(err)
		}
		close(done)
	})
	<-done
}
