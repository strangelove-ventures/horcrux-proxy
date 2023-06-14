package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/strangelove-ventures/horcrux-proxy/config"
)

const (
	flagNode    = "node"
	flagAddress = "address"
)

func configCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Commands to configure the horcrux signer",
	}

	cmd.AddCommand(initCmd(a))

	return cmd
}

func initCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"i"},
		Short:   "initialize configuration file",
		Long:    "initialize configuration file and home directory if one doesn't already exist",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cmdFlags := cmd.Flags()

			address, _ := cmdFlags.GetString(flagAddress)
			nodes, _ := cmdFlags.GetStringSlice(flagNode)

			cn, err := config.ChainNodesFromFlag(nodes)
			if err != nil {
				return err
			}

			// silence usage after all input has been validated
			cmd.SilenceUsage = true

			// create the config file
			a.config.Config = config.Config{
				ListenAddr: address,
				ChainNodes: cn,
			}

			if err := os.MkdirAll(filepath.Dir(a.config.ConfigFile), 0700); err != nil {
				return err
			}

			if err = a.config.WriteConfigFile(); err != nil {
				return err
			}

			fmt.Printf("Successfully initialized configuration: %s\n", a.config.ConfigFile)
			return nil
		},
	}

	f := cmd.Flags()

	f.StringP(flagAddress, "a", "tcp://0.0.0.0:1234", "listen address in format tcp://0.0.0.0:1234")
	f.StringSliceP(flagNode, "n", []string{}, "chain nodes in format tcp://{node-addr}:{privval-port} \n"+
		"(e.g. --node tcp://sentry-1:1234 --node tcp://sentry-2:1234 --node tcp://sentry-3:1234 )")

	return cmd
}
