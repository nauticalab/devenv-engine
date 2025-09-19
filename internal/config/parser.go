package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadGlobalConfig loads the global configuration file (devenv.yaml) from the config directory.
// Returns a BaseConfig pre-populated with system defaults. If the global config file exists,
// YAML values override the defaults. If the file doesn't exist, returns defaults without error.
func LoadGlobalConfig(configDir string) (*BaseConfig, error) {
	globalConfigPath := filepath.Join(configDir, "devenv.yaml")

	// Start with system defaults
	globalConfig := NewBaseConfigWithDefaults()

	// Check if global config file exists
	if _, err := os.Stat(globalConfigPath); os.IsNotExist(err) {
		return &globalConfig, nil // Return defaults if file doesn't exist
	}

	// Read the global config file
	data, err := os.ReadFile(globalConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config file %s: %w", globalConfigPath, err)
	}

	// Unmarshal into pre-populated struct - only overrides present fields
	if err := yaml.Unmarshal(data, &globalConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in global config %s: %w", globalConfigPath, err)
	}

	return &globalConfig, nil
}

// LoadDeveloperConfig loads and parses a developer's configuration file
// from the specified directory. It reads the devenv-config.yaml file from
// the developer's subdirectory, validates the configuration, and returns
// a populated DevEnvConfig struct with only basic validation.
//
// This function does NOT merge with global defaults - use LoadDeveloperConfigWithGlobalDefaults
// for that functionality.
func LoadDeveloperConfig(configDir, developerName string) (*DevEnvConfig, error) {
	developerDir := filepath.Join(configDir, developerName)
	configPath := filepath.Join(developerDir, "devenv-config.yaml")

	// Check if the config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Create empty config (no defaults)
	var config DevEnvConfig

	// Parse the YAML
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", configPath, err)
	}

	// Basic validation
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", configPath, err)
	}

	config.DeveloperDir = developerDir

	return &config, nil
}

// LoadDeveloperConfigWithGlobalDefaults loads a developer config and merges it with global defaults.
// This is the recommended loading function that provides the complete configuration hierarchy:
// System defaults → Global config → User config
func LoadDeveloperConfigWithGlobalDefaults(configDir, developerName string) (*DevEnvConfig, error) {
	// Step 1: Load global configuration (pre-populated with system defaults)
	globalConfig, err := LoadGlobalConfig(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	// Step 2: Create user config pre-populated with global config values
	userConfig := &DevEnvConfig{
		BaseConfig: *globalConfig, // Copy all global values (which include system defaults)
	}

	// Step 3: Load user YAML
	developerDir := filepath.Join(configDir, developerName)
	configPath := filepath.Join(developerDir, "devenv-config.yaml")

	// Check if the config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", configPath)
	}

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Step 4: Unmarshal user YAML - overwrites only fields present in YAML
	if err := yaml.Unmarshal(data, userConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", configPath, err)
	}

	// Step 5: Merge additive list fields (packages, volumes, SSH keys)
	userConfig.mergeListFields(globalConfig)

	// Step 6: Set developer directory and validate
	userConfig.DeveloperDir = developerDir

	if err := validateConfig(userConfig); err != nil {
		return nil, fmt.Errorf("invalid configuration in %s: %w", configPath, err)
	}

	return userConfig, nil
}

// mergeListFields handles additive merging for packages, volumes, and SSH keys
func (config *DevEnvConfig) mergeListFields(globalConfig *BaseConfig) {
	// Save current user values before merging
	userPackagesPython := config.Packages.Python
	userPackagesAPT := config.Packages.APT
	userVolumes := config.Volumes

	// Merge packages: global packages + user packages
	config.Packages.Python = mergeStringSlices(globalConfig.Packages.Python, userPackagesPython)
	config.Packages.APT = mergeStringSlices(globalConfig.Packages.APT, userPackagesAPT)

	// Merge volumes: global volumes + user volumes
	config.Volumes = mergeVolumes(globalConfig.Volumes, userVolumes)

	// Merge SSH keys: global SSH keys + user SSH keys
	globalSSHKeys, err := globalConfig.GetSSHKeys()
	if err != nil {
		globalSSHKeys = []string{}
	}

	userSSHKeys, err := config.GetSSHKeys()
	if err != nil {
		userSSHKeys = []string{}
	}

	mergedSSHKeys := mergeStringSlices(globalSSHKeys, userSSHKeys)
	config.SSHPublicKey = mergedSSHKeys
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
	sshKeys, err := config.GetSSHKeys()
	if err != nil {
		return fmt.Errorf("invalid sshPublicKey format: %w", err)
	}

	if len(sshKeys) == 0 {
		return fmt.Errorf("at least one SSH public key is required")
	}

	return nil
}

// ============================================================================
// Utility functions for configuration merging and normalization
// ============================================================================

// mergeStringSlices combines two string slices, removing duplicates
// The global slice items come first, followed by user slice items
func mergeStringSlices(global, user []string) []string {
	if len(global) == 0 {
		return user
	}
	if len(user) == 0 {
		return global
	}

	// Use map to track seen values and maintain order
	seen := make(map[string]bool)
	var result []string

	// Add global values first
	for _, item := range global {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	// Add user values, skipping duplicates
	for _, item := range user {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// mergeVolumes combines global and user volume mounts
// User volumes with the same name override global volumes
func mergeVolumes(global, user []VolumeMount) []VolumeMount {
	if len(global) == 0 {
		return user
	}
	if len(user) == 0 {
		return global
	}

	// Create map of user volumes by name for quick lookup
	userVolumesByName := make(map[string]VolumeMount)
	for _, vol := range user {
		userVolumesByName[vol.Name] = vol
	}

	var result []VolumeMount

	// Add global volumes, but skip if user has same name
	for _, globalVol := range global {
		if _, exists := userVolumesByName[globalVol.Name]; !exists {
			result = append(result, globalVol)
		}
	}

	// Add all user volumes
	result = append(result, user...)

	return result
}

// normalizeSSHKeys converts the flexible SSH key field to a string slice
// Handles both single string and string array formats from YAML
func normalizeSSHKeys(sshKeyField any) ([]string, error) {
	if sshKeyField == nil {
		return []string{}, nil
	}

	switch keys := sshKeyField.(type) {
	case string:
		// Single SSH key
		if keys == "" {
			return []string{}, fmt.Errorf("SSH key cannot be empty string")
		}
		return []string{keys}, nil

	case []interface{}:
		// Array of SSH keys (from YAML)
		var result []string
		for i, key := range keys {
			keyStr, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("SSH key at index %d is not a string", i)
			}
			if keyStr == "" {
				return nil, fmt.Errorf("SSH key at index %d cannot be empty", i)
			}
			result = append(result, keyStr)
		}
		if len(result) == 0 {
			return nil, fmt.Errorf("SSH key array cannot be empty")
		}
		return result, nil

	case []string:
		// Direct string slice
		if len(keys) == 0 {
			return nil, fmt.Errorf("SSH key array cannot be empty")
		}
		for i, key := range keys {
			if key == "" {
				return nil, fmt.Errorf("SSH key at index %d cannot be empty", i)
			}
		}
		return keys, nil

	default:
		return nil, fmt.Errorf("SSH key field must be string or array of strings, got %T", sshKeyField)
	}
}
