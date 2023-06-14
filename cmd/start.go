package cmd

import (
	"fmt"
	"net"

	"github.com/cometbft/cometbft/crypto/ed25519"
	cometlog "github.com/cometbft/cometbft/libs/log"
	cometnet "github.com/cometbft/cometbft/libs/net"
	cometos "github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/libs/service"
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

			logger := cometlog.NewTMLogger(cometlog.NewSyncWriter(out)).With("module", "validator")

			logger.Info(
				"Horcrux Proxy",
			)

			listener := newSignerListenerEndpoint(logger, a.config.Config.ListenAddr)

			services, err := signer.StartRemoteSigners(logger, listener, a.config.Config.Nodes())
			if err != nil {
				return fmt.Errorf("failed to start remote signer(s): %w", err)
			}

			waitAndTerminate(logger, services)

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

func waitAndTerminate(logger cometlog.Logger, services []service.Service) {
	done := make(chan struct{})

	cometos.TrapSignal(logger, func() {
		for _, service := range services {
			err := service.Stop()
			if err != nil {
				panic(err)
			}
		}
		close(done)
	})
	<-done
}
