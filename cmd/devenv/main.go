package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Build-time variables (will be set by build system later)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

var (
	// Global flags (avaialble to all commands)
	verbose bool

	// Command-specific flags
	output  string
	dryRun  bool
	allDevs bool
)

// Root command
var rootCmd = &cobra.Command{
	Use:   "devenv",
	Short: "Generate Kubernetes manifests for developer environments",
	Long: `DevENV generates Kubernetes manifests from simple YAML configurations.

It processes developer environmnet configurations and generates complete
Kubernets resources including StatefulSets, Services, Ingresses, and ConfigMaps.`,
}

var generateCmd = &cobra.Command{
	Use:   "generate [developer-name]",
	Short: "Generate manifests for a developer environment",
	Long: `Generate Kubernetes manifests for a specific developer or all developers.

Examples:
  devenv generate eywalker
  devenv generate --all-developers --output ./manifests`,
	Args: cobra.MaximumNArgs(1), // At max 1 argument
	Run: func(cmd *cobra.Command, args []string) {
		//Validation logic
		if allDevs && len(args) > 0 {
			fmt.Fprintf(os.Stderr, "error: Cannot specify developer name with --all-developers flag\n")
			os.Exit(1)
		}

		if !allDevs && len(args) == 0 {
			fmt.Fprintf(os.Stderr, "Error: Please specify a developer name or use --all-developers\n")
			cmd.Help()
			os.Exit(1)
		}

		// Execute the logic (placeholder for now)
		if allDevs {
			fmt.Println("Generating manifests for all developers...")
			if verbose {
				fmt.Printf("Output directory: %s\n", output)
			}
			// TODO: implement all developers logic
		} else {
			developerName := args[0]
			fmt.Printf("Generating manifests for developer: %s\n", developerName)
			if verbose {
				fmt.Printf("Output directory: %s\n", output)
				fmt.Printf("Dry run mode: %t\n", dryRun)
			}
			// TODO: implement single developer logic
		}
	},
}

// Version subcommand
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("devenv version %s\n", version)

		if verbose {
			fmt.Printf("  Build time: %s\n", buildTime)
			fmt.Printf("  Git commit: %s\n", gitCommit)
		}
	},
}

func init() {
	// Add subcommands to root
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(versionCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Generate command specific flags
	generateCmd.Flags().StringVarP(&output, "output", "o", "./build", "Output directory for generated manifests")
	generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without creating files")
	generateCmd.Flags().BoolVar(&allDevs, "all-developers", false, "Generate manifests for all developers")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
