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
	Use:   "manager",
	Short: "Manage developer environments in Kubernetes",
	Long: `DevENV Manager provides tools for managing and inspecting developer 
environments running in Kubernetes clusters.

Use this tool to list pods, check statuses, and perform administrative tasks
on developer environments.`,
}

func init() {
	// Global flags available to all subcommands
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add subcommands to root
	rootCmd.AddCommand(podsCmd)
	rootCmd.AddCommand(versionCmd)
}
