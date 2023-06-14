package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/strangelove-ventures/horcrux-proxy/config"
	"gopkg.in/yaml.v2"
)

type appState struct {
	config config.RuntimeConfig
}

func rootCmd(a *appState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "horcrux-proxy",
		Short: "A tendermint remote signer proxy",
	}

	cmd.AddCommand(configCmd(a))
	cmd.AddCommand(startCmd(a))
	cmd.AddCommand(versionCmd())

	cmd.PersistentFlags().StringVar(
		&a.config.HomeDir,
		"home",
		"",
		"Directory for config and data (default is $HOME/.horcrux-proxy)",
	)

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	a := &appState{}

	cobra.OnInitialize(func() { initConfig(a) })

	if err := rootCmd(a).Execute(); err != nil {
		// Cobra will print the error
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig(a *appState) {
	var home string
	if a.config.HomeDir == "" {
		userHome, err := homedir.Dir()
		handleInitError(err)
		home = filepath.Join(userHome, ".horcrux-proxy")
	} else {
		home = a.config.HomeDir
	}
	a.config = config.RuntimeConfig{
		HomeDir:    home,
		ConfigFile: filepath.Join(home, "config.yaml"),
	}
	viper.SetConfigFile(a.config.ConfigFile)
	viper.SetEnvPrefix("horcrux-proxy")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("no config exists at default location", err)
		return
	}
	handleInitError(viper.Unmarshal(&a.config.Config))
	bz, err := os.ReadFile(viper.ConfigFileUsed())
	handleInitError(err)
	handleInitError(yaml.Unmarshal(bz, &a.config.Config))
}

func handleInitError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
