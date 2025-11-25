package main

import (
	"github.com/nauticalab/devenv-engine/internal/cli"
	"github.com/spf13/cobra"
)

var (
	// Validate command flags
	validateConfigDir string
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate [developer-name]",
	Short: "Validate developer environment configurations",
	Long: `Validate developer environment configurations for common issues.

This command checks for:
- SSH port conflicts between developers
- SSH ports outside valid NodePort range (30000-32767)
- Missing or invalid configuration files

Examples:
  devenv validate                    # Validate all configurations
  devenv validate eywalker          # Validate specific developer (includes conflict checking)
  devenv validate --config-dir ./configs`,
	Args: cobra.MaximumNArgs(1), // At most 1 argument (developer name)
	Run: func(cmd *cobra.Command, args []string) {
		opts := cli.ValidateOptions{
			ConfigDir: validateConfigDir,
			Verbose:   verbose,
		}

		if len(args) == 0 {
			// Validate all developers
			cli.ValidateRunAll(opts)
		} else {
			// Validate single developer (with conflict checking)
			developerName := args[0]
			cli.ValidateRunSingle(developerName, opts)
		}
	},
}

func init() {
	// Validate command specific flags
	validateCmd.Flags().StringVar(&validateConfigDir, "config-dir", "./developers", "Directory containing developer configuration files")
}
