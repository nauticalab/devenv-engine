package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walkerlab/devenv-engine/internal/validation"
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
		validator := validation.NewPortValidator(validateConfigDir)

		if len(args) == 0 {
			// Validate all developers
			validateAll(validator)
		} else {
			// Validate single developer (with conflict checking)
			developerName := args[0]
			validateSingle(validator, developerName)
		}
	},
}

func init() {
	// Validate command specific flags
	validateCmd.Flags().StringVar(&validateConfigDir, "config-dir", "./developers", "Directory containing developer configuration files")
}

// validateAll validates all developer configurations
func validateAll(validator *validation.PortValidator) {
	fmt.Println("🔍 Validating all developer configurations...")

	result, err := validator.ValidateAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	printValidationResult(result, "")

	if !result.IsValid {
		os.Exit(1)
	}
}

// validateSingle validates a single developer configuration (including conflicts)
func validateSingle(validator *validation.PortValidator, developerName string) {
	fmt.Printf("🔍 Validating configuration for developer: %s\n", developerName)

	result, err := validator.ValidateSingle(developerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	printValidationResult(result, developerName)

	if !result.IsValid {
		os.Exit(1)
	}
}

// printValidationResult prints the validation results in a user-friendly format
func printValidationResult(result *validation.ValidationResult, targetUser string) {
	// Print warnings first
	for _, warning := range result.Warnings {
		fmt.Printf("⚠️  Warning: %s\n", warning.Message)
		if warning.FilePath != "" && verbose {
			fmt.Printf("   File: %s\n", warning.FilePath)
		}
	}

	// Print errors with context-specific messaging
	for _, err := range result.Errors {
		switch err.Type {
		case "conflict":
			if targetUser != "" {
				// Single user validation - show from their perspective
				fmt.Printf("❌ Port Conflict: %s\n", err.Message)
			} else {
				// All users validation - show general conflict
				fmt.Printf("❌ Port Conflict: Port %d is assigned to multiple developers: %v\n", err.Port, err.Users)
			}
			if verbose {
				fmt.Printf("   Affected users: %v\n", err.Users)
			}
		case "out_of_range":
			fmt.Printf("❌ Invalid Port Range: %s\n", err.Message)
			if verbose && err.FilePath != "" {
				fmt.Printf("   File: %s\n", err.FilePath)
			}
		case "invalid":
			fmt.Printf("❌ Configuration Error: %s\n", err.Message)
			if verbose && err.FilePath != "" {
				fmt.Printf("   File: %s\n", err.FilePath)
			}
		default:
			fmt.Printf("❌ Error: %s\n", err.Message)
		}
	}

	// Print summary
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		if targetUser != "" {
			fmt.Printf("✅ Configuration for %s is valid!\n", targetUser)
		} else {
			fmt.Println("✅ All configurations are valid!")
		}
	} else if result.IsValid {
		if targetUser != "" {
			fmt.Printf("✅ Configuration for %s is valid (%d warnings)\n", targetUser, len(result.Warnings))
		} else {
			fmt.Printf("✅ All configurations are valid (%d warnings)\n", len(result.Warnings))
		}
	} else {
		fmt.Printf("❌ Validation failed with %d errors and %d warnings\n", len(result.Errors), len(result.Warnings))

		// Provide helpful suggestions
		if len(result.Errors) > 0 {
			fmt.Println("\n💡 Suggestions:")
			hasConflicts := false
			hasRangeErrors := false

			for _, err := range result.Errors {
				if err.Type == "conflict" && !hasConflicts {
					if targetUser != "" {
						fmt.Printf("   • Assign a unique SSH port to %s\n", targetUser)
					} else {
						fmt.Println("   • Assign unique SSH ports to each developer")
					}
					fmt.Printf("   • Valid port range: %d-%d\n", validation.NodePortMin, validation.NodePortMax)
					hasConflicts = true
				}
				if err.Type == "out_of_range" && !hasRangeErrors {
					fmt.Printf("   • Use ports between %d and %d (Kubernetes NodePort range)\n", validation.NodePortMin, validation.NodePortMax)
					hasRangeErrors = true
				}
			}
		}
	}
}
