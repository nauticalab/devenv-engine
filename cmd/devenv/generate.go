package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walkerlab/devenv-engine/internal/config"
	"github.com/walkerlab/devenv-engine/internal/templates"
)

var (
	// Command-specific flags for generate
	outputDir string
	configDir string // Input directory for developer configs
	dryRun    bool
	allDevs   bool
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

		// Execute the logic (placeholder for now)
		if allDevs {
			fmt.Println("Generating manifests for all developers...")
			if verbose {
				fmt.Printf("Output directory: %s\n", outputDir)
			}
			// TODO: implement all developers logic
		} else {
			developerName := args[0]
			generateSingleDeveloper(developerName)
		}
	},
}

func init() {
	// Generate command specific flags
	generateCmd.Flags().StringVarP(&outputDir, "output", "o", "./build", "Output directory for generated manifests")
	generateCmd.Flags().StringVar(&configDir, "config-dir", "./developers", "Directory containing developer configuration files")
	generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be generated without creating files")
	generateCmd.Flags().BoolVar(&allDevs, "all-developers", false, "Generate manifests for all developers")

}

// generateSingleDeveloper handles generation for a single developer
func generateSingleDeveloper(developerName string) {
	fmt.Printf("Generating manifests for developer: %s\n", developerName)

	if verbose {
		fmt.Printf("Output directory: %s\n", outputDir)
		fmt.Printf("Config directory: %s\n", configDir)
		fmt.Printf("Dry run mode: %t\n", dryRun)
	}

	cfg, err := config.LoadDeveloperConfigWithGlobals(configDir, developerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config for developer %s: %v\n", developerName, err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully loaded configuration for developer: %s\n", cfg.Name)

	if verbose {
		printConfigSummary(cfg)
	}

	if !dryRun {
		if err := generateManifests(cfg, outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating manifests: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("🔍 Dry run - would generate manifests to: %s\n", outputDir)
	}
}

// generateManifests creates Kubernetes manifests for a developer
func generateManifests(cfg *config.DevEnvConfig, outputDir string) error {
	// Create template renderer
	renderer := templates.NewRenderer(outputDir)

	// Render all main templates
	if err := renderer.RenderAll(cfg); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	fmt.Printf("🎉 Successfully generated manifests for %s\n", cfg.Name)

	return nil
}

// Helper function to print config summary
func printConfigSummary(cfg *config.DevEnvConfig) {
	fmt.Printf("\nConfiguration Summary:\n")
	fmt.Printf("  Name: %s\n", cfg.Name)

	sshKeys, _ := cfg.GetSSHKeys()
	fmt.Printf("  SSH Keys: %d configured\n", len(sshKeys))

	if cfg.SSHPort != 0 {
		fmt.Printf("  SSH Port: %d\n", cfg.SSHPort)
	}

	if cfg.Git.Name != "" {
		fmt.Printf("  Git: %s <%s>\n", cfg.Git.Name, cfg.Git.Email)
	}

	if cfg.Resources.CPU != nil || cfg.Resources.Memory != "" {
		cpuStr := formatCPU(cfg.Resources.CPU)
		fmt.Printf("  Resources: CPU=%s, Memory=%s, GPU=%d\n",
			cpuStr, cfg.Resources.Memory, cfg.Resources.GPU)

	}

	if len(cfg.Volumes) > 0 {
		fmt.Printf("  Volumes: %d configured\n", len(cfg.Volumes))
	}

	fmt.Printf("  Developer Config Dir: %s\n", cfg.GetDeveloperDir())
}

// Helper function to format CPU value for display
func formatCPU(cpu any) string {
	if cpu == nil {
		return "default"
	}
	switch v := cpu.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return fmt.Sprintf("%v", v) // Fallback
	}
}
