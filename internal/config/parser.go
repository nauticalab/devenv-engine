package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadDeveloperConfig loads and parses a developer's configuration file
func LoadDeveloperConfig(developerName string) (*DevEnvConfig, error) {
	configPath := filepath.Join("developers", developerName, "devenv-config.yaml")

	// Check if the config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse the YAML
	var config DevEnvConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", configPath, err)
	}

	// Basic validation
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configurationin %s: %w", configPath, err)
	}
	return &config, nil

}

// validateConfig performs basic validation on the configuration
func validateConfig(config *DevEnvConfig) error {
	if config.Name == "" {
		return fmt.Errorf("name field is required")
	}

	if config.SSHPublicKey == nil {
		return fmt.Errorf("sshPublicKey field is required")
	}

	// Validate SSH keys format
	sshKeys, err := normalizeSSHKeys(config.SSHPublicKey)
	if err != nil {
		return fmt.Errorf("invalid sshPublicKey format: %w", err)
	}

	if len(sshKeys) == 0 {
		return fmt.Errorf("at least one SSH public key is required")
	}

	return nil
}

// normalizeSSHKeys converts the flexible SSH key field to a string slice
func normalizeSSHKeys(sshKeyField any) ([]string, error) {
	switch keys := sshKeyField.(type) {
	case string:
		// Single SSH key as string
		if keys == "" {
			return nil, fmt.Errorf("SSH key cannot be empty")
		}
		return []string{keys}, nil
	case []any:
		// Multiple SSH keys as interface slice
		result := make([]string, len(keys))
		for i, key := range keys {
			keyStr, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("SSH key at index %d is not a string", i)
			}
			if keyStr == "" {
				return nil, fmt.Errorf("SSH key at index %d cannot be empty", i)
			}
			result[i] = keyStr
		}
		return result, nil
	case []string:
		// Already a string slice (shouldn't happen with YAML parsing, but just in case)
		for i, key := range keys {
			if key == "" {
				return nil, fmt.Errorf("SSH key at index %d cannot be empty", i)
			}
		}
		return keys, nil
	default:
		return nil, fmt.Errorf("SSH key field must be either a string or array of strings")
	}
}

// GetSSHKeys returns the SSH keys as a normalized string slice
func (c *DevEnvConfig) GetSSHKeys() ([]string, error) {
	return normalizeSSHKeys(c.SSHPublicKey)
}
