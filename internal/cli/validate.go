package cli

import (
	"fmt"
	"os"

	"github.com/nauticalab/devenv-engine/internal/validation"
)

// Options holds configuration for the validate command
type ValidateOptions struct {
	ConfigDir string
	Verbose   bool
}

// RunAll validates all developer configurations
func ValidateRunAll(opts ValidateOptions) {
	fmt.Println("🔍 Validating all developer configurations...")

	validator := validation.NewPortValidator(opts.ConfigDir)
	result, err := validator.ValidateAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	printValidationResult(result, "", opts.Verbose)

	if !result.IsValid {
		os.Exit(1)
	}
}

// RunSingle validates a single developer configuration (including conflicts)
func ValidateRunSingle(developerName string, opts ValidateOptions) {
	fmt.Printf("🔍 Validating configuration for developer: %s\n", developerName)

	validator := validation.NewPortValidator(opts.ConfigDir)
	result, err := validator.ValidateSingle(developerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	printValidationResult(result, developerName, opts.Verbose)

	if !result.IsValid {
		os.Exit(1)
	}
}

// printValidationResult prints the validation results in a user-friendly format
func printValidationResult(result *validation.ValidationResult, targetUser string, verbose bool) {
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
