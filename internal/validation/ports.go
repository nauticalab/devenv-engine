// Package validation provides logic for validating developer configurations.
// It includes checks for port conflicts, valid port ranges, and other configuration rules.
package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nauticalab/devenv-engine/internal/config"
)

const (
	// Kubernetes NodePort valid range
	NodePortMin = 30000
	NodePortMax = 32767
)

// PortValidator handles SSH port validation across developer configurations
type PortValidator struct {
	// configDir is the directory containing developer configurations
	configDir string
}

// ValidationResult contains all validation results
type ValidationResult struct {
	// Errors is a list of fatal validation errors
	Errors []ValidationError
	// Warnings is a list of non-fatal validation warnings
	Warnings []ValidationWarning
	// IsValid indicates if the validation passed (no errors)
	IsValid bool
}

// ValidationError represents a validation failure
type ValidationError struct {
	// Type is the category of error (e.g., "conflict", "out_of_range", "invalid")
	Type string
	// Port is the port number involved in the error (if applicable)
	Port int
	// Users is a list of developers involved in the error
	Users []string
	// Message is a human-readable error description
	Message string
	// FilePath is the path to the configuration file causing the error
	FilePath string
}

// ValidationWarning represents a non-fatal validation issue
type ValidationWarning struct {
	// Type is the category of warning
	Type string
	// User is the developer associated with the warning
	User string
	// Message is a human-readable warning description
	Message string
	// FilePath is the path to the configuration file
	FilePath string
}

// NewPortValidator creates a new port validator
func NewPortValidator(configDir string) *PortValidator {
	return &PortValidator{configDir: configDir}
}

// ValidateAll scans all developer configs and validates SSH ports
func (pv *PortValidator) ValidateAll() (*ValidationResult, error) {
	result := &ValidationResult{
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		IsValid:  true,
	}

	// Find all developer configuration directories
	developers, err := pv.findDeveloperDirs()
	if err != nil {
		return nil, fmt.Errorf("failed to scan developer directories in %s: %w", pv.configDir, err)
	}
	if len(developers) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "no_configs",
			Message: fmt.Sprintf("No developer configurations found in %s", pv.configDir),
		})
		return result, nil
	}

	// Load all configurations and collect port assignments
	portAssignments := make(map[int][]string) // port -> []users
	for _, developerName := range developers {
		port, validationError, validationWarning := pv.validateSingleDeveloper(developerName)
		if validationError != nil {
			result.Errors = append(result.Errors, *validationError)
			result.IsValid = false
			continue
		}
		if validationWarning != nil {
			result.Warnings = append(result.Warnings, *validationWarning)
			continue
		}

		// Track port assignments for conflict detection
		portAssignments[port] = append(portAssignments[port], developerName)
	}

	// Check for port conflicts
	for port, users := range portAssignments {
		if len(users) > 1 {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "conflict",
				Port:    port,
				Users:   users,
				Message: fmt.Sprintf("Port %d is assigned to multiple developers: %s", port, strings.Join(users, ", ")),
			})
			result.IsValid = false
		}
	}

	return result, nil
}

func (pv *PortValidator) validateSingleDeveloper(developerName string) (int, *ValidationError, *ValidationWarning) {
	cfg, err := config.LoadDeveloperConfig(pv.configDir, developerName)
	if err != nil {
		return 0, &ValidationError{
			Type:     "invalid",
			Users:    []string{developerName},
			Message:  fmt.Sprintf("Failed to load config: %v", err),
			FilePath: filepath.Join(pv.configDir, developerName, "devenv-config.yaml"),
		}, nil
	}

	// Check if SSH port is configured
	if cfg.SSHPort == 0 {
		return 0, nil, &ValidationWarning{
			Type:     "no_ssh_port",
			User:     developerName,
			Message:  fmt.Sprintf("No SSH port configured for developer %s", developerName),
			FilePath: filepath.Join(pv.configDir, developerName, "devenv-config.yaml"),
		}
	}

	// Validate port range
	if cfg.SSHPort < NodePortMin || cfg.SSHPort > NodePortMax {
		return cfg.SSHPort, &ValidationError{
			Type:     "out_of_range",
			Port:     cfg.SSHPort,
			Users:    []string{developerName},
			Message:  fmt.Sprintf("SSH port %d for developer %s is out of valid range (%d-%d)", cfg.SSHPort, developerName, NodePortMin, NodePortMax),
			FilePath: filepath.Join(pv.configDir, developerName, "devenv-config.yaml"),
		}, nil
	}

	return cfg.SSHPort, nil, nil
}

// ValidateSingle validates a single developer by running full validation and filtering results
func (pv *PortValidator) ValidateSingle(developerName string) (*ValidationResult, error) {
	// Run full validation to catch all conflicts
	fullResult, err := pv.ValidateAll()
	if err != nil {
		return nil, err
	}

	// Filter results to only those relevant to the target developer
	result := &ValidationResult{
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
		IsValid:  true,
	}

	// Filter errors - include if target user is involved
	for _, err := range fullResult.Errors {
		if pv.errorInvolvesUser(err, developerName) {
			result.Errors = append(result.Errors, err)
			result.IsValid = false
		}
	}

	// Filter warnings - include if it's about the target user
	for _, warning := range fullResult.Warnings {
		if warning.User == developerName {
			result.Warnings = append(result.Warnings, warning)
		}
	}

	return result, nil
}

// errorInvolvesUser checks if a validation error involves the specified user
func (pv *PortValidator) errorInvolvesUser(err ValidationError, targetUser string) bool {
	for _, user := range err.Users {
		if user == targetUser {
			return true
		}
	}
	return false
}

func (pv *PortValidator) findDeveloperDirs() ([]string, error) {
	var developers []string

	entries, err := os.ReadDir(pv.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory %s: %w", pv.configDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check if devenv-config.yaml exists in the directory
			configPath := filepath.Join(pv.configDir, entry.Name(), "devenv-config.yaml")
			if _, err := os.Stat(configPath); err == nil {
				developers = append(developers, entry.Name())
			}
		}
	}

	return developers, nil
}
