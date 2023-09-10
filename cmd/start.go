package cmd

import (
	"fmt"
	"net"

	"github.com/cometbft/cometbft/crypto/ed25519"
	cometlog "github.com/cometbft/cometbft/libs/log"
	cometnet "github.com/cometbft/cometbft/libs/net"
	cometos "github.com/cometbft/cometbft/libs/os"
	"github.com/spf13/cobra"

	"github.com/strangelove-ventures/horcrux-proxy/privval"
	"github.com/strangelove-ventures/horcrux-proxy/signer"
)

const (
	flagListen = "listen"
	flagAll    = "all"
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

			a.logger.Info("Horcrux Proxy")

			addr, _ := cmd.Flags().GetString(flagListen)
			all, _ := cmd.Flags().GetBool(flagAll)

			a.listener = newSignerListenerEndpoint(a.logger, addr)

			if err := a.listener.Start(); err != nil {
				return fmt.Errorf("failed to start listener: %w", err)
			}

			a.sentries = make(map[string]*signer.ReconnRemoteSigner)

			if err := watchForChangedSentries(cmd.Context(), a, all); err != nil {
				return err
			}

			waitAndTerminate(a)

			return nil
		},
	}

	cmd.Flags().StringP(flagListen, "l", "tcp://0.0.0.0:1234", "Privval listen address for the proxy")
	cmd.Flags().BoolP(flagAll, "a", false, "Connect to sentries on all nodes")

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
