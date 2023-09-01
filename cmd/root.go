package cmd

import (
	"os"

	cometlog "github.com/cometbft/cometbft/libs/log"
	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/horcrux-proxy/privval"
	"github.com/strangelove-ventures/horcrux-proxy/signer"
)

type appState struct {
	logger   cometlog.Logger
	listener *privval.SignerListenerEndpoint
	sentries map[string]*signer.ReconnRemoteSigner
}

func rootCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "horcrux-proxy",
		Short: "A tendermint remote signer proxy",
	}

	cmd.AddCommand(startCmd(a))
	cmd.AddCommand(versionCmd())

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd(new(appState)).Execute(); err != nil {
		// Cobra will print the error
		os.Exit(1)
	}
}
