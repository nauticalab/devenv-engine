package main

import (
	"github.com/spf13/cobra"
)

var (
	// Global flags (available to all commands)
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "devenv",
	Short: "Generate Kubernetes manifests for developer environments",
	Long: `DevENV generates Kubernetes manifests from simple YAML configurations.

It processes developer environment configurations and generates complete
Kubernetes resources including StatefulSets, Services, Ingresses, and ConfigMaps.`,
}

func init() {
	// Global flags available to all subcommands
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add subcommands to root
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(validateCmd)
}
