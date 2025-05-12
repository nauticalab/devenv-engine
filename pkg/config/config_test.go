package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSystemConfig(t *testing.T) {
	// Create a temporary config file
	content := `
organization:
  name: "Test Org"
  domain: "test.org"
defaults:
  resources:
    cpu: "2"
    memory: "8Gi"
    storage: "20Gi"
    gpu: 0
  packages:
    python:
      - numpy
      - pandas
    apt:
      - curl
      - wget
  image: "test-image:latest"
portManagement:
  sshPortRange:
    start: 30000
    end: 32000
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "system-config.yaml")

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the config
	config, err := LoadSystemConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load system config: %v", err)
	}

	// Check values
	if config.Organization.Name != "Test Org" {
		t.Errorf("Expected org name 'Test Org', got '%s'", config.Organization.Name)
	}

	if config.Defaults.Resources.CPU != "2" {
		t.Errorf("Expected CPU '2', got '%s'", config.Defaults.Resources.CPU)
	}

	if config.Defaults.Image != "test-image:latest" {
		t.Errorf("Expected image 'test-image:latest', got '%s'", config.Defaults.Image)
	}

	if len(config.Defaults.Packages.Python) != 2 || config.Defaults.Packages.Python[0] != "numpy" {
		t.Errorf("Expected Python packages [numpy pandas], got %v", config.Defaults.Packages.Python)
	}
}

func TestMergeConfigurations(t *testing.T) {
	// Create a simple test setup
	sysConfig := &SystemConfig{
		Defaults: Defaults{
			Resources: Resources{
				CPU:     "2",
				Memory:  "8Gi",
				Storage: "20Gi",
				GPU:     0,
			},
			Packages: Packages{
				Python: []string{"numpy", "pandas"},
				Apt:    []string{"curl", "wget"},
			},
		},
		Profiles: map[string]Profile{
			"test-profile": {
				Name: "test-profile",
				Resources: Resources{
					CPU:    "4",
					Memory: "16Gi",
					GPU:    1,
				},
				Packages: Packages{
					Python: []string{"tensorflow"},
				},
			},
		},
	}

	userConfig := &UserConfig{
		Name:         "testuser",
		SSHPublicKey: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC test-key"},
		UID:          1000,
		IsAdmin:      false,
		Defaults: UserDefaults{
			Git: GitConfig{
				Name:  "Test User",
				Email: "test@example.com",
			},
		},
	}

	envConfig := &EnvironmentConfig{
		Name:    "testenv",
		Profile: "test-profile",
		Resources: Resources{
			Storage: "50Gi",
		},
		Packages: Packages{
			Python: []string{"pytorch"},
		},
	}

	// Merge the configurations
	result, err := MergeConfigurations(sysConfig, userConfig, envConfig)
	if err != nil {
		t.Fatalf("Failed to merge configurations: %v", err)
	}

	// Check the result
	if result.Name != "testuser-testenv" {
		t.Errorf("Expected name 'testuser-testenv', got '%s'", result.Name)
	}

	if result.Resources.CPU != "4" {
		t.Errorf("Expected CPU '4', got '%s'", result.Resources.CPU)
	}

	if result.Resources.Memory != "16Gi" {
		t.Errorf("Expected Memory '16Gi', got '%s'", result.Resources.Memory)
	}

	if result.Resources.Storage != "50Gi" {
		t.Errorf("Expected Storage '50Gi', got '%s'", result.Resources.Storage)
	}

	if result.Resources.GPU != 1 {
		t.Errorf("Expected GPU 1, got %d", result.Resources.GPU)
	}

	// Check packages
	// Check packages
	expectedPythonPackages := []string{"numpy", "pandas", "tensorflow", "pytorch"}
	if len(result.Packages.Python) != len(expectedPythonPackages) {
		t.Errorf("Expected %d Python packages, got %d: %v",
			len(expectedPythonPackages), len(result.Packages.Python), result.Packages.Python)
	}

	// Verify all expected packages are present
	for _, pkg := range expectedPythonPackages {
		found := false
		for _, resultPkg := range result.Packages.Python {
			if resultPkg == pkg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected Python package %s not found in result: %v", pkg, result.Packages.Python)
		}
	}
	// Check user settings
	if result.GitConfig.Name != "Test User" {
		t.Errorf("Expected Git name 'Test User', got '%s'", result.GitConfig.Name)
	}
}
