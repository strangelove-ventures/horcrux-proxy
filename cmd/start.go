package cmd

import (
	"fmt"
	"net"
	"time"

	"github.com/cometbft/cometbft/crypto/ed25519"
	cometlog "github.com/cometbft/cometbft/libs/log"
	cometnet "github.com/cometbft/cometbft/libs/net"
	cometos "github.com/cometbft/cometbft/libs/os"
	"github.com/spf13/cobra"

	"github.com/strangelove-ventures/horcrux-proxy/privval"
	"github.com/strangelove-ventures/horcrux-proxy/signer"
)

func startCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "start",
		Short:        "Start horcrux-proxy process",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			a.logger = cometlog.NewTMLogger(cometlog.NewSyncWriter(out)).With("module", "validator")

			a.logger.Info(
				"Horcrux Proxy",
			)

			a.listener = newSignerListenerEndpoint(a.logger, a.config.Config.ListenAddr)

			if err := a.listener.Start(); err != nil {
				return fmt.Errorf("failed to start listener: %w", err)
			}

			a.sentries = make(map[string]*signer.ReconnRemoteSigner)

			for _, node := range a.config.Config.ChainNodes {
				// CometBFT requires a connection within 3 seconds of start or crashes
				// A long timeout such as 30 seconds would cause the sentry to fail in loops
				// Use a short timeout and dial often to connect within 3 second window
				dialer := net.Dialer{Timeout: 2 * time.Second}
				s := signer.NewReconnRemoteSigner(node.PrivValAddr, a.logger, a.listener, dialer)

				if err := s.Start(); err != nil {
					return fmt.Errorf("failed to start remote signer(s): %w", err)
				}
				a.sentries[node.PrivValAddr] = s
			}

			go watchForConfigFileUpdates(cmd.Context(), a)

			waitAndTerminate(a)

			return nil
		},
	}

	return cmd
}

func newSignerListenerEndpoint(logger cometlog.Logger, addr string) *privval.SignerListenerEndpoint {
	proto, address := cometnet.ProtocolAndAddress(addr)

	ln, err := net.Listen(proto, address)
	logger.Info("SignerListener: Listening", "proto", proto, "address", address)
	if err != nil {
		panic(err)
	}

	var listener net.Listener

	if proto == "unix" {
		unixLn := privval.NewUnixListener(ln)
		listener = unixLn
	} else {
		tcpLn := privval.NewTCPListener(ln, ed25519.GenPrivKey())
		listener = tcpLn
	}

	return privval.NewSignerListenerEndpoint(
		logger,
		listener,
	)
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
		if err := a.listener.Stop(); err != nil {
			panic(err)
		}
		close(done)
	})
	<-done
}
