package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadUserConfig loads a developer's main configuration
func LoadUserConfig(path string) (*UserConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read user config file: %w", err)
	}

	var config UserConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse user config: %w", err)
	}

	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("user config must specify a name")
	}

	if len(config.SSHPublicKey) == 0 {
		return nil, fmt.Errorf("user config must specify at least one SSH public key")
	}

	// Set defaults
	if config.UID == 0 {
		config.UID = 1000 // Default UID
	}

	return &config, nil
}

// LoadEnvironmentConfig loads a environment-specific configuration
func LoadEnvironmentConfig(path string) (*EnvironmentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read environment config file: %w", err)
	}
	var config EnvironmentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse environment config: %w", err)
	}
	// Validate required fields
	if config.Name == "" {
		return nil, fmt.Errorf("environment config must specify a name")
	}

	return &config, nil
}

// GetUserEnvironment lists all environments for a user
func GetUserEnvironment(userDir string, userConfig *UserConfig) ([]string, error) {
	// Start with environments listed in the user config
	envs := make(map[string]bool)
	for _, env := range userConfig.Environments {
		envs[env] = true
	}

	// Look for environment files in the environments directory
	envsDir := filepath.Join(userDir, "environments")
	if _, err := os.Stat(envsDir); err == nil {
		files, err := os.ReadDir(envsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read environments directory: %w", err)
		}

		for _, entry := range files {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if filepath.Ext(name) == ".yaml" || filepath.Ext(name) == ".yml" {
				// Add the environment name (without extension)
				envName := name[:len(name)-len(filepath.Ext(name))]
				envs[envName] = true
			}
		}
	} else if len(envs) != 0 {
		return nil, fmt.Errorf("failed to locate the environments directory: %w", err)
	}

	// Convert map keys to slice
	result := make([]string, 0, len(envs))
	for env := range envs {
		result = append(result, env)
	}

	return result, nil
}
