package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var outputFormat string

var appCtx *App

var rootCmd = &cobra.Command{
	Use:   "zavictl",
	Short: "zavictl is an open-source CLI-first DevOps control plane",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		app, err := BootstrapApp()
		if err != nil {
			return err
		}
		appCtx = app
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.zavictl.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format (text, json, yaml)")
	rootCmd.PersistentFlags().String("environment", "", "Target environment (e.g. staging, production)")
	rootCmd.PersistentFlags().String("project", "", "Project context for this command")
	rootCmd.PersistentFlags().Bool("quiet", false, "Suppress all non-essential output")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug output with internal details")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable color output")
	rootCmd.PersistentFlags().Duration("timeout", 0, "Timeout for this command (e.g. 30s, 5m)")
	rootCmd.PersistentFlags().Bool("dry-run", false, "Simulate the command without making changes")
	rootCmd.PersistentFlags().Bool("non-interactive", false, "Disable all prompts; fail if input is required")
	// --yes suppresses confirmation prompts ONLY — it never bypasses policy checks or approval gates
	rootCmd.PersistentFlags().Bool("yes", false, "Automatically confirm prompts (does not bypass policy or approval gates)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".zavictl")
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
