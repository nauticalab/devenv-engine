package main

import (
	"fmt"
	"os"

	"github.com/nauticalab/devenv-engine/internal/cli"
	"github.com/spf13/cobra"
)

var (
	// Command-specific flags for generate
	outputDir           string
	configDir           string // Input directory for developer configs
	dryRun              bool
	allDevs             bool
	skipSystemManifests bool
)

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

		opts := cli.GenerateOptions{
			OutputDir:           outputDir,
			ConfigDir:           configDir,
			DryRun:              dryRun,
			SkipSystemManifests: skipSystemManifests,
		}

		// Execute the logic
		if allDevs {
			fmt.Println("Generating manifests for all developers...")
			cli.GenerateRunAll(opts)
		} else {
			developerName := args[0]
			cli.GenerateRunSingle(developerName, opts)
		}
	},
}

func init() {
	// Generate command specific flags
	generateCmd.Flags().StringVarP(&outputDir, "output", "o", "./build", "Output directory for generated manifests")
	generateCmd.Flags().StringVar(&configDir, "config-dir", "./developers", "Directory containing developer configuration files")
	generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without creating files")
	generateCmd.Flags().BoolVar(&allDevs, "all-developers", false, "Generate manifests for all developers")
	generateCmd.Flags().BoolVar(&skipSystemManifests, "skip-system-manifests", false, "Skip generating system manifests")

}
