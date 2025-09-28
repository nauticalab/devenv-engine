package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGlobalConfig(t *testing.T) {
	t.Run("global config file exists", func(t *testing.T) {
		// Create temp directory and global config file
		tempDir := t.TempDir()
		globalConfigPath := filepath.Join(tempDir, "devenv.yaml")

		globalConfigYAML := `image: "custom:latest"
installHomebrew: false
packages:
  apt: ["curl", "git"]
  python: ["requests"]
resources:
  cpu: 4
  memory: "16Gi"
`
		err := os.WriteFile(globalConfigPath, []byte(globalConfigYAML), 0644)
		require.NoError(t, err)

		// Load global config
		config, err := LoadGlobalConfig(tempDir)
		require.NoError(t, err)

		// Test that YAML values override defaults
		assert.Equal(t, "custom:latest", config.Image)
		assert.False(t, config.InstallHomebrew) // Override default true
		assert.Equal(t, 4, config.Resources.CPU)
		assert.Equal(t, "16Gi", config.Resources.Memory)
		assert.Equal(t, []string{"curl", "git"}, config.Packages.APT)
		assert.Equal(t, []string{"requests"}, config.Packages.Python)

		// Test that unspecified values keep defaults
		assert.True(t, config.ClearLocalPackages == false)     // Default
		assert.True(t, config.ClearVSCodeCache == false)       // Default
		assert.Equal(t, "/opt/venv/bin", config.PythonBinPath) // Default
		assert.Equal(t, 1000, config.UID)                      // Default
	})

	t.Run("global config file does not exist", func(t *testing.T) {
		// Create empty temp directory
		tempDir := t.TempDir()

		// Load global config from non-existent file
		config, err := LoadGlobalConfig(tempDir)
		require.NoError(t, err)

		// Should return all defaults
		assert.Equal(t, "ubuntu:22.04", config.Image)
		assert.True(t, config.InstallHomebrew)
		assert.False(t, config.ClearLocalPackages)
		assert.False(t, config.ClearVSCodeCache)
		assert.Equal(t, "/opt/venv/bin", config.PythonBinPath)
		assert.Equal(t, 1000, config.UID)
		assert.Equal(t, 2, config.Resources.CPU)
		assert.Equal(t, "8Gi", config.Resources.Memory)
		assert.Equal(t, []string{}, config.Packages.APT)
		assert.Equal(t, []string{}, config.Packages.Python)
	})

	t.Run("invalid YAML in global config", func(t *testing.T) {
		tempDir := t.TempDir()
		globalConfigPath := filepath.Join(tempDir, "devenv.yaml")

		// Write invalid YAML
		invalidYAML := `image: "test
installHomebrew: [invalid`
		err := os.WriteFile(globalConfigPath, []byte(invalidYAML), 0644)
		require.NoError(t, err)

		// Should return error
		_, err = LoadGlobalConfig(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse YAML")
	})
}

func TestLoadDeveloperConfig(t *testing.T) {
	t.Run("valid developer config", func(t *testing.T) {
		tempDir := t.TempDir()
		developerDir := filepath.Join(tempDir, "alice")
		err := os.MkdirAll(developerDir, 0755)
		require.NoError(t, err)

		configPath := filepath.Join(developerDir, "devenv-config.yaml")
		configYAML := `name: alice
sshPublicKey:
  - "ssh-rsa AAAAB3NzaC1yc2E alice@example.com"
sshPort: 30022
isAdmin: true
git:
  name: "Alice Smith"
  email: "alice@example.com"
packages:
  python: ["numpy", "pandas"]
  apt: ["vim"]
`
		err = os.WriteFile(configPath, []byte(configYAML), 0644)
		require.NoError(t, err)

		// Load developer config
		config, err := LoadDeveloperConfig(tempDir, "alice")
		require.NoError(t, err)

		// Test basic fields
		assert.Equal(t, "alice", config.Name)
		assert.Equal(t, 30022, config.SSHPort)
		assert.True(t, config.IsAdmin)
		assert.Equal(t, "Alice Smith", config.Git.Name)
		assert.Equal(t, "alice@example.com", config.Git.Email)
		assert.Equal(t, []string{"numpy", "pandas"}, config.Packages.Python)
		assert.Equal(t, []string{"vim"}, config.Packages.APT)

		// Test SSH keys
		sshKeys, err := config.GetSSHKeys()
		require.NoError(t, err)
		assert.Equal(t, []string{"ssh-rsa AAAAB3NzaC1yc2E alice@example.com"}, sshKeys)

		// Test developer directory is set
		assert.Equal(t, developerDir, config.DeveloperDir)
	})

	t.Run("config file not found", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := LoadDeveloperConfig(tempDir, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration file not found")
	})

	t.Run("invalid config - missing name", func(t *testing.T) {
		tempDir := t.TempDir()
		developerDir := filepath.Join(tempDir, "alice")
		err := os.MkdirAll(developerDir, 0755)
		require.NoError(t, err)

		configPath := filepath.Join(developerDir, "devenv-config.yaml")
		configYAML := `sshPublicKey: "ssh-rsa AAAAB3NzaC1yc2E alice@example.com"`
		err = os.WriteFile(configPath, []byte(configYAML), 0644)
		require.NoError(t, err)

		_, err = LoadDeveloperConfig(tempDir, "alice")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'Name' is required")
	})

	t.Run("invalid config - missing SSH key", func(t *testing.T) {
		tempDir := t.TempDir()
		developerDir := filepath.Join(tempDir, "alice")
		err := os.MkdirAll(developerDir, 0755)
		require.NoError(t, err)

		configPath := filepath.Join(developerDir, "devenv-config.yaml")
		configYAML := `name: alice`
		err = os.WriteFile(configPath, []byte(configYAML), 0644)
		require.NoError(t, err)

		_, err = LoadDeveloperConfig(tempDir, "alice")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SSH public key is required")
	})
}

func TestLoadDeveloperConfigWithGlobalDefaults(t *testing.T) {
	t.Run("complete integration - global and user config", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create global config
		globalConfigYAML := `image: "global:latest"
installHomebrew: true
clearLocalPackages: true
packages:
  apt: ["curl", "git"]
  python: ["requests"]
resources:
  cpu: 4
  memory: "16Gi"
sshPublicKey: "ssh-rsa AAAAB3NzaC1yc2E admin@company.com"
`
		globalConfigPath := filepath.Join(tempDir, "devenv.yaml")
		err := os.WriteFile(globalConfigPath, []byte(globalConfigYAML), 0644)
		require.NoError(t, err)

		// Create user config
		developerDir := filepath.Join(tempDir, "alice")
		err = os.MkdirAll(developerDir, 0755)
		require.NoError(t, err)

		userConfigYAML := `name: alice
sshPublicKey: "ssh-rsa AAAAB3NzaC1yc2E alice@example.com"
installHomebrew: false
packages:
  apt: ["vim"]
  python: ["pandas"]
git:
  name: "Alice Smith"
  email: "alice@example.com"
`
		configPath := filepath.Join(developerDir, "devenv-config.yaml")
		err = os.WriteFile(configPath, []byte(userConfigYAML), 0644)
		require.NoError(t, err)

		// Load with global defaults
		config, err := LoadDeveloperConfigWithGlobalDefaults(tempDir, "alice")
		require.NoError(t, err)

		// Test user-specific fields
		assert.Equal(t, "alice", config.Name)
		assert.Equal(t, "Alice Smith", config.Git.Name)
		assert.Equal(t, "alice@example.com", config.Git.Email)

		// Test override fields (user overrides global)
		assert.Equal(t, "global:latest", config.Image)   // User didn't specify, uses global
		assert.False(t, config.InstallHomebrew)          // User overrides global true with false
		assert.True(t, config.ClearLocalPackages)        // User didn't specify, uses global
		assert.Equal(t, 4, config.Resources.CPU)         // User didn't specify, uses global
		assert.Equal(t, "16Gi", config.Resources.Memory) // User didn't specify, uses global

		// Test additive fields (global + user)
		expectedAPT := []string{"curl", "git", "vim"} // Global + user packages
		assert.Equal(t, expectedAPT, config.Packages.APT)

		expectedPython := []string{"requests", "pandas"} // Global + user packages
		assert.Equal(t, expectedPython, config.Packages.Python)

		// Test SSH key merging (global + user)
		sshKeys, err := config.GetSSHKeys()
		require.NoError(t, err)
		expectedSSHKeys := []string{
			"ssh-rsa AAAAB3NzaC1yc2E admin@company.com", // Global
			"ssh-rsa AAAAB3NzaC1yc2E alice@example.com", // User
		}
		assert.Equal(t, expectedSSHKeys, sshKeys)

		// Test developer directory is set
		assert.Equal(t, developerDir, config.DeveloperDir)
	})

	t.Run("user config with no global config", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create only user config (no global config file)
		developerDir := filepath.Join(tempDir, "alice")
		err := os.MkdirAll(developerDir, 0755)
		require.NoError(t, err)

		userConfigYAML := `name: alice
sshPublicKey: "ssh-rsa AAAAB3NzaC1yc2E alice@example.com"
installHomebrew: false
`
		configPath := filepath.Join(developerDir, "devenv-config.yaml")
		err = os.WriteFile(configPath, []byte(userConfigYAML), 0644)
		require.NoError(t, err)

		// Load with global defaults
		config, err := LoadDeveloperConfigWithGlobalDefaults(tempDir, "alice")
		require.NoError(t, err)

		// Should get system defaults + user overrides
		assert.Equal(t, "alice", config.Name)
		assert.Equal(t, "ubuntu:22.04", config.Image)          // System default
		assert.False(t, config.InstallHomebrew)                // User override
		assert.False(t, config.ClearLocalPackages)             // System default
		assert.Equal(t, "/opt/venv/bin", config.PythonBinPath) // System default
	})
}

func TestMergeListFields(t *testing.T) {
	t.Run("merge packages", func(t *testing.T) {
		globalConfig := &BaseConfig{
			Packages: PackageConfig{
				APT:    []string{"curl", "git"},
				Python: []string{"requests"},
			},
		}

		userConfig := &DevEnvConfig{
			BaseConfig: BaseConfig{
				Packages: PackageConfig{
					APT:    []string{"vim", "curl"}, // "curl" is duplicate
					Python: []string{"pandas"},
				},
			},
		}

		userConfig.mergeListFields(globalConfig)

		expectedAPT := []string{"curl", "git", "vim"} // Deduplication
		expectedPython := []string{"requests", "pandas"}

		assert.Equal(t, expectedAPT, userConfig.Packages.APT)
		assert.Equal(t, expectedPython, userConfig.Packages.Python)
	})

	t.Run("merge volumes", func(t *testing.T) {
		globalConfig := &BaseConfig{
			Volumes: []VolumeMount{
				{Name: "data", LocalPath: "/global/data", ContainerPath: "/data"},
				{Name: "logs", LocalPath: "/global/logs", ContainerPath: "/logs"},
			},
		}

		userConfig := &DevEnvConfig{
			BaseConfig: BaseConfig{
				Volumes: []VolumeMount{
					{Name: "data", LocalPath: "/user/data", ContainerPath: "/data"}, // Same name - user overrides
					{Name: "cache", LocalPath: "/user/cache", ContainerPath: "/cache"},
				},
			},
		}

		userConfig.mergeListFields(globalConfig)

		expected := []VolumeMount{
			{Name: "logs", LocalPath: "/global/logs", ContainerPath: "/logs"},  // From global (not overridden)
			{Name: "data", LocalPath: "/user/data", ContainerPath: "/data"},    // User overrides global
			{Name: "cache", LocalPath: "/user/cache", ContainerPath: "/cache"}, // User only
		}

		assert.ElementsMatch(t, expected, userConfig.Volumes)
	})

	t.Run("merge SSH keys", func(t *testing.T) {
		globalConfig := &BaseConfig{
			SSHPublicKey: []string{"ssh-rsa AAAAB3... admin@company.com"},
		}

		userConfig := &DevEnvConfig{
			BaseConfig: BaseConfig{
				SSHPublicKey: []string{"ssh-rsa AAAAB3... alice@example.com"},
			},
		}

		userConfig.mergeListFields(globalConfig)

		sshKeys, err := userConfig.GetSSHKeys()
		require.NoError(t, err)

		expected := []string{
			"ssh-rsa AAAAB3... admin@company.com", // Global
			"ssh-rsa AAAAB3... alice@example.com", // User
		}
		assert.Equal(t, expected, sshKeys)
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &DevEnvConfig{
			Name: "alice",
			BaseConfig: BaseConfig{
				SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2E alice@example.com",
			},
		}

		err := ValidateDevEnvConfig(config)
		assert.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		config := &DevEnvConfig{
			BaseConfig: BaseConfig{
				SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2E... alice@example.com",
			},
		}

		err := ValidateDevEnvConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'Name' is required")
	})

	t.Run("missing SSH public key", func(t *testing.T) {
		config := &DevEnvConfig{
			Name: "alice",
		}

		err := ValidateDevEnvConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SSH public key is required")
	})

	t.Run("invalid SSH key format", func(t *testing.T) {
		config := &DevEnvConfig{
			Name: "alice",
			BaseConfig: BaseConfig{
				SSHPublicKey: 123, // Invalid type
			},
		}

		err := ValidateDevEnvConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SSH key format")
	})

	t.Run("empty SSH key", func(t *testing.T) {
		config := &DevEnvConfig{
			Name: "alice",
			BaseConfig: BaseConfig{
				SSHPublicKey: "",
			},
		}

		err := ValidateDevEnvConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SSH key format")
	})
}

// Test utility functions
func TestMergeStringSlices(t *testing.T) {
	tests := []struct {
		name     string
		global   []string
		user     []string
		expected []string
	}{
		{
			name:     "no duplicates",
			global:   []string{"a", "b"},
			user:     []string{"c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "with duplicates",
			global:   []string{"a", "b"},
			user:     []string{"b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty global",
			global:   []string{},
			user:     []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			name:     "empty user",
			global:   []string{"a", "b"},
			user:     []string{},
			expected: []string{"a", "b"},
		},
		{
			name:     "both empty",
			global:   []string{},
			user:     []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeStringSlices(tt.global, tt.user)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeVolumes(t *testing.T) {
	global := []VolumeMount{
		{Name: "data", LocalPath: "/global/data", ContainerPath: "/data"},
		{Name: "logs", LocalPath: "/global/logs", ContainerPath: "/logs"},
	}
	user := []VolumeMount{
		{Name: "data", LocalPath: "/user/data", ContainerPath: "/data"}, // Override
		{Name: "cache", LocalPath: "/user/cache", ContainerPath: "/cache"},
	}

	result := mergeVolumes(global, user)

	// User "data" should override global "data", "logs" should remain, "cache" should be added
	expected := []VolumeMount{
		{Name: "logs", LocalPath: "/global/logs", ContainerPath: "/logs"},
		{Name: "data", LocalPath: "/user/data", ContainerPath: "/data"},
		{Name: "cache", LocalPath: "/user/cache", ContainerPath: "/cache"},
	}

	assert.ElementsMatch(t, expected, result)
}

func TestNormalizeSSHKeys(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expected    []string
		expectError bool
	}{
		{
			name:        "single string",
			input:       "ssh-rsa AAAAB3... user@host",
			expected:    []string{"ssh-rsa AAAAB3... user@host"},
			expectError: false,
		},
		{
			name:        "string array",
			input:       []string{"key1", "key2"},
			expected:    []string{"key1", "key2"},
			expectError: false,
		},
		{
			name:        "interface array",
			input:       []interface{}{"key1", "key2"},
			expected:    []string{"key1", "key2"},
			expectError: false,
		},
		{
			name:        "nil input",
			input:       nil,
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty array",
			input:       []string{},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid type",
			input:       123,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeSSHKeys(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
