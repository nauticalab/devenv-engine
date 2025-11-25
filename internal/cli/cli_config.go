package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CLIConfig represents the configuration for the CLI
type CLIConfig struct {
	ManagerURL  string `yaml:"managerURL"`
	SATokenPath string `yaml:"saTokenPath"`
}

// LoadCLIConfig loads configuration from multiple sources in order of precedence:
// 1. Flags (handled by caller)
// 2. Environment variables
// 3. Config file (~/.devenv/config.yaml)
func LoadCLIConfig() (*CLIConfig, error) {
	config := &CLIConfig{}

	// 1. Load from config file
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".devenv", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			if err == nil {
				if err := yaml.Unmarshal(data, config); err != nil {
					return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
				}
			}
		}
	}

	// 2. Load from environment variables (override config file)
	if envURL := os.Getenv("DEVEN_MANAGER_URL"); envURL != "" {
		config.ManagerURL = envURL
	}

	return config, nil
}
