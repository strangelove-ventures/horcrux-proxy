package cmd

import (
	"fmt"
	"log/slog"

	cometos "github.com/cometbft/cometbft/libs/os"
	"github.com/spf13/cobra"

	"github.com/strangelove-ventures/horcrux-proxy/privval"
	"github.com/strangelove-ventures/horcrux-proxy/signer"
)

const (
	flagLogLevel    = "log-level"
	flagListen      = "listen"
	flagAll         = "all"
	flagGRPCAddress = "grpc"
	flagOperator    = "operator"
	flagSentry      = "sentry"
	flagSentryLabel = "label"
	flagMaxReadSize = "max-read-size"
)

func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "start",
		Short:        "Start horcrux-proxy process",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			logLevel, _ := cmd.Flags().GetString(flagLogLevel)

			var slogLevel slog.Leveler
			switch logLevel {
			case "debug":
				slogLevel = slog.LevelDebug
			case "info":
				slogLevel = slog.LevelInfo
			case "error":
				slogLevel = slog.LevelError
			case "warn":
				slogLevel = slog.LevelWarn
			}

			logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{
				Level: slogLevel,
			}))
			logger.Info("Horcrux Proxy")

			listenAddrs, _ := cmd.Flags().GetStringArray(flagListen)
			all, _ := cmd.Flags().GetBool(flagAll)

			listeners := make([]privval.SignerListener, len(listenAddrs))
			for i, addr := range listenAddrs {
				listeners[i] = privval.NewSignerListener(logger, addr)
			}

			var hc signer.HorcruxConnection

			grpcAddr, _ := cmd.Flags().GetString(flagGRPCAddress)

			var err error

			if grpcAddr != "" {
				hc, err = signer.NewHorcruxGRPCClient(logger, grpcAddr)
				if err != nil {
					return fmt.Errorf("failed to create grpc connection: %w", err)
				}
			} else {
				loadBalancer := privval.NewRemoteSignerLoadBalancer(logger, listeners)
				if err = loadBalancer.Start(); err != nil {
					return fmt.Errorf("failed to start listener(s): %w", err)
				}
				defer logIfErr(logger, loadBalancer.Stop)

				hc = loadBalancer
			}

			ctx := cmd.Context()

			// if we're running in kubernetes, we can auto-discover sentries
			operator, _ := cmd.Flags().GetBool(flagOperator)
			sentries, _ := cmd.Flags().GetStringArray(flagSentry)
			labels, _ := cmd.Flags().GetStringArray(flagSentryLabel)
			maxReadSize, _ := cmd.Flags().GetInt(flagMaxReadSize)

			watcher, err := NewSentryWatcher(ctx, logger, labels, all, hc, operator, sentries, maxReadSize)
			if err != nil {
				return err
			}
			defer logIfErr(logger, watcher.Stop)
			go watcher.Watch(ctx, maxReadSize)

			waitForSignals(logger)

			return nil
		},
	}

	cmd.Flags().StringArrayP(flagListen, "l", nil, "Privval listen addresses for the proxy (e.g. tcp://0.0.0.0:1234)")
	cmd.Flags().StringArrayP(flagSentry, "s", nil, "Privval connect addresses for the proxy")
	cmd.Flags().StringArrayP(flagSentryLabel, "L", nil, "the label of the sentry to connect to")
	cmd.Flags().BoolP(flagOperator, "o", true, "Use this when running in kubernetes with the Cosmos Operator to auto-discover sentries")
	cmd.Flags().StringP(flagGRPCAddress, "g", "", "GRPC address for the proxy")
	cmd.Flags().BoolP(flagAll, "a", false, "Connect to sentries on all nodes")
	cmd.Flags().String(flagLogLevel, "info", "Set log level (debug, info, error, none)")
	cmd.Flags().Int(flagMaxReadSize, 1024*1024, "Max read size for privval messages")

	return cmd
}

func logIfErr(logger *slog.Logger, fn func() error) {
	if err := fn(); err != nil {
		logger.Error("Error", "err", err)
	}
}

func waitForSignals(logger *slog.Logger) {
	done := make(chan struct{})
	cometos.TrapSignal(logger, func() {
		close(done)
	})
	<-done
}
