package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "horcrux-proxy",
		Short: "A tendermint remote signer proxy",
	}

	cmd.AddCommand(startCmd())
	cmd.AddCommand(versionCmd())

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd().Execute(); err != nil {
		// Cobra will print the error
		os.Exit(1)
	}
}
