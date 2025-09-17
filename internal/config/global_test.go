// Save this as: internal/config/global_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGlobalConfig(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create global config file
	globalConfigContent := `
# Global DevEnv Defaults for Organization
image: "ubuntu:22.04"
uid: 1000
sshPublicKey:
  - "ssh-rsa AAAAB3NzaC1yc2EA...admin1"
  - "ssh-rsa AAAAB3NzaC1yc2EB...admin2"
resources:
  cpu: 4
  memory: "16Gi" 
  storage: "100Gi"
packages:
  apt:
    - vim
    - git 
    - curl
  python:
    - requests
    - numpy
volumes:
  - name: shared
    localPath: /mnt/shared
    containerPath: /shared
`

	globalConfigPath := filepath.Join(tempDir, "devenv.yaml")
	require.NoError(t, os.WriteFile(globalConfigPath, []byte(globalConfigContent), 0644))

	// Test loading global config
	globalConfig, err := LoadGlobalConfig(tempDir)
	require.NoError(t, err)
	require.NotNil(t, globalConfig)

	// Verify global config values
	assert.Equal(t, "ubuntu:22.04", globalConfig.Image)
	assert.Equal(t, 1000, globalConfig.UID)
	assert.Equal(t, 4, globalConfig.Resources.CPU)
	assert.Equal(t, "16Gi", globalConfig.Resources.Memory)
	assert.Equal(t, []string{"vim", "git", "curl"}, globalConfig.Packages.APT)
	assert.Equal(t, []string{"requests", "numpy"}, globalConfig.Packages.Python)
	assert.Len(t, globalConfig.Volumes, 1)
	assert.Equal(t, "shared", globalConfig.Volumes[0].Name)

	// Test SSH keys
	sshKeys, err := globalConfig.GetSSHKeys()
	require.NoError(t, err)
	expectedKeys := []string{
		"ssh-rsa AAAAB3NzaC1yc2EA...admin1",
		"ssh-rsa AAAAB3NzaC1yc2EB...admin2",
	}
	assert.ElementsMatch(t, expectedKeys, sshKeys)
}

func TestLoadGlobalConfig_NotFound(t *testing.T) {
	// Test when global config doesn't exist
	tempDir := t.TempDir()

	globalConfig, err := LoadGlobalConfig(tempDir)

	// Should return empty global config, not error
	require.NoError(t, err)
	require.NotNil(t, globalConfig)

	// Should be essentially empty
	assert.Empty(t, globalConfig.Image)
	assert.Equal(t, 0, globalConfig.UID)
	assert.Empty(t, globalConfig.Packages.APT)
	assert.Empty(t, globalConfig.Packages.Python)
	assert.Empty(t, globalConfig.Volumes)
}
func TestMergeConfigs(t *testing.T) {
	t.Run("User overrides with global defaults present", func(t *testing.T) {
		// Create global config with defaults
		globalConfig := &GlobalConfig{
			Image: "ubuntu:22.04",
			UID:   1000,
			SSHPublicKey: []string{
				"ssh-rsa AAAAB3NzaC1...admin1",
				"ssh-rsa AAAAB3NzaC1...admin2",
			},
			Resources: ResourceConfig{
				CPU:     4,
				Memory:  "16Gi",
				Storage: "100Gi",
				GPU:     1,
			},
			Packages: PackageConfig{
				APT:    []string{"vim", "git"},
				Python: []string{"requests"},
			},
			Volumes: []VolumeMount{
				{Name: "shared", LocalPath: "/mnt/shared", ContainerPath: "/shared"},
			},
		}

		// Create user config that overrides some values
		userConfig := &DevEnvConfig{
			Name:         "alice",
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1...alice",
			Resources: ResourceConfig{
				CPU: 8, // Override global CPU
				GPU: 2, // Override global GPU
				// Memory and Storage not specified - should get global defaults
			},
			Packages: PackageConfig{
				APT:    []string{"tmux"},   // Add to global APT packages
				Python: []string{"pandas"}, // Add to global Python packages
			},
			Volumes: []VolumeMount{
				{Name: "personal", LocalPath: "/home/alice", ContainerPath: "/home"},
			},
		}

		// Merge configurations
		merged := MergeConfigs(globalConfig, userConfig)

		// Test user overrides
		assert.Equal(t, 8, merged.Resources.CPU) // User overrides global
		assert.Equal(t, 2, merged.Resources.GPU) // User overrides global
		assert.Equal(t, "alice", merged.Name)    // User-specific field

		// Test global defaults used when user doesn't specify
		assert.Equal(t, "16Gi", merged.Resources.Memory)   // From global
		assert.Equal(t, "100Gi", merged.Resources.Storage) // From global
		assert.Equal(t, "ubuntu:22.04", merged.Image)      // From global
		assert.Equal(t, 1000, merged.UID)                  // From global

		// Verify the methods return correct values
		assert.Equal(t, "8", merged.CPU())          // User override
		assert.Equal(t, "16Gi", merged.Memory())    // Global default
		assert.Equal(t, "1000", merged.GetUserID()) // Global default

		// Test additive fields
		expectedAPT := []string{"vim", "git", "tmux"}
		assert.ElementsMatch(t, expectedAPT, merged.Packages.APT)

		expectedPython := []string{"requests", "pandas"}
		assert.ElementsMatch(t, expectedPython, merged.Packages.Python)

		// Test SSH key merging
		mergedKeys, err := merged.GetSSHKeys()
		require.NoError(t, err)
		expectedSSHKeys := []string{
			"ssh-rsa AAAAB3NzaC1...admin1", // Global
			"ssh-rsa AAAAB3NzaC1...admin2", // Global
			"ssh-rsa AAAAB3NzaC1...alice",  // User
		}
		assert.ElementsMatch(t, expectedSSHKeys, mergedKeys)
	})

	t.Run("No user override - global defaults used", func(t *testing.T) {
		// Global config with all resource defaults
		globalConfig := &GlobalConfig{
			Image: "ubuntu:22.04",
			UID:   2000,
			Resources: ResourceConfig{
				CPU:     6,
				Memory:  "32Gi",
				Storage: "200Gi",
				GPU:     1,
			},
		}

		// User config with minimal required fields only
		userConfig := &DevEnvConfig{
			Name:         "bob",
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1...bob",
			// No resource overrides - should get all global defaults
		}

		merged := MergeConfigs(globalConfig, userConfig)

		// All resources should come from global config
		assert.Equal(t, 6, merged.Resources.CPU)           // From global
		assert.Equal(t, "32Gi", merged.Resources.Memory)   // From global
		assert.Equal(t, "200Gi", merged.Resources.Storage) // From global
		assert.Equal(t, 1, merged.Resources.GPU)           // From global
		assert.Equal(t, "ubuntu:22.04", merged.Image)      // From global
		assert.Equal(t, 2000, merged.UID)                  // From global

		// Verify methods return global values
		assert.Equal(t, "6", merged.CPU())          // Global default
		assert.Equal(t, "32Gi", merged.Memory())    // Global default
		assert.Equal(t, "2000", merged.GetUserID()) // Global default

		// User-specific fields preserved
		assert.Equal(t, "bob", merged.Name)
	})

	t.Run("Neither user nor global specified - SystemDefaults used", func(t *testing.T) {
		// Empty global config
		globalConfig := &GlobalConfig{
			// No defaults specified
		}

		// Minimal user config
		userConfig := &DevEnvConfig{
			Name:         "charlie",
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1...charlie",
			// No resources, image, or UID specified
		}

		merged := MergeConfigs(globalConfig, userConfig)

		// Resources should be empty/zero in merged config
		assert.Nil(t, merged.Resources.CPU)           // Not set
		assert.Equal(t, "", merged.Resources.Memory)  // Not set
		assert.Equal(t, "", merged.Resources.Storage) // Not set
		assert.Equal(t, 0, merged.Resources.GPU)      // Not set
		assert.Equal(t, "", merged.Image)             // Not set
		assert.Equal(t, 0, merged.UID)                // Not set

		// But methods should return SystemDefaults
		assert.Equal(t, "2", merged.CPU())          // SystemDefaults.CPU
		assert.Equal(t, "8Gi", merged.Memory())     // SystemDefaults.Memory
		assert.Equal(t, "1000", merged.GetUserID()) // SystemDefaults.UID

		// User-specific fields preserved
		assert.Equal(t, "charlie", merged.Name)
	})

	t.Run("Mixed scenarios - some fields from each tier", func(t *testing.T) {
		// Global config with partial defaults
		globalConfig := &GlobalConfig{
			Image: "ubuntu:22.04", // Global default
			UID:   1500,           // Global default
			Resources: ResourceConfig{
				CPU:    4,      // Global default
				Memory: "16Gi", // Global default
				// Storage not specified - will fall back to system default
				GPU: 1, // Global default
			},
		}

		// User config with selective overrides
		userConfig := &DevEnvConfig{
			Name:         "diana",
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1...diana",
			Resources: ResourceConfig{
				CPU: 12, // User override
				// Memory not specified - should get global default
				// Storage not specified - should get system default (empty)
				// GPU not specified - should get global default
			},
			// Image not specified - should get global default
			// UID not specified - should get global default
		}

		merged := MergeConfigs(globalConfig, userConfig)

		// Verify three-tier hierarchy
		assert.Equal(t, 12, merged.Resources.CPU)        // User override
		assert.Equal(t, "16Gi", merged.Resources.Memory) // Global default
		assert.Equal(t, "", merged.Resources.Storage)    // Neither specified
		assert.Equal(t, 1, merged.Resources.GPU)         // Global default
		assert.Equal(t, "ubuntu:22.04", merged.Image)    // Global default
		assert.Equal(t, 1500, merged.UID)                // Global default

		// Verify method outputs
		assert.Equal(t, "12", merged.CPU())         // User override
		assert.Equal(t, "16Gi", merged.Memory())    // Global default
		assert.Equal(t, "1500", merged.GetUserID()) // Global default
	})

	t.Run("Edge cases - zero values and empty strings", func(t *testing.T) {
		// Test that explicit zero/empty values are preserved vs. unspecified values
		globalConfig := &GlobalConfig{
			UID: 1000,
			Resources: ResourceConfig{
				CPU:    4,
				Memory: "16Gi",
				GPU:    2,
			},
		}

		userConfig := &DevEnvConfig{
			Name:         "eve",
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1...eve",
			UID:          0, // Explicit zero - should this override global or fall back?
			Resources: ResourceConfig{
				CPU:    0,  // Explicit zero - should this override global?
				Memory: "", // Explicit empty - should this override global?
				GPU:    0,  // Explicit zero - should this override global?
			},
		}

		merged := MergeConfigs(globalConfig, userConfig)

		// Current behavior: explicit zero/empty values don't override global defaults
		// This is the expected behavior for your current implementation
		assert.Equal(t, 1000, merged.UID)                // Global used (0 treated as unspecified)
		assert.Equal(t, 4, merged.Resources.CPU)         // Global used (nil != 0)
		assert.Equal(t, "16Gi", merged.Resources.Memory) // Global used ("" treated as unspecified)
		assert.Equal(t, 2, merged.Resources.GPU)         // Global used (0 treated as unspecified)

		// Note: This behavior might need discussion - should explicit zero override?
	})

	t.Run("Global config with duplicate user SSH keys", func(t *testing.T) {
		globalConfig := &GlobalConfig{
			SSHPublicKey: []string{
				"ssh-rsa AAAAB3NzaC1...admin1",
				"ssh-rsa AAAAB3NzaC1...admin2",
			},
		}

		userConfig := &DevEnvConfig{
			Name: "frank",
			SSHPublicKey: []string{
				"ssh-rsa AAAAB3NzaC1...admin1", // Duplicate of global key
				"ssh-rsa AAAAB3NzaC1...frank",  // Unique user key
			},
		}

		merged := MergeConfigs(globalConfig, userConfig)

		// Should deduplicate SSH keys
		mergedKeys, err := merged.GetSSHKeys()
		require.NoError(t, err)

		expectedKeys := []string{
			"ssh-rsa AAAAB3NzaC1...admin1", // Should appear only once
			"ssh-rsa AAAAB3NzaC1...admin2", // Global key
			"ssh-rsa AAAAB3NzaC1...frank",  // User key
		}
		assert.ElementsMatch(t, expectedKeys, mergedKeys)
		assert.Len(t, mergedKeys, 3) // Ensure no duplicates
	})
}
