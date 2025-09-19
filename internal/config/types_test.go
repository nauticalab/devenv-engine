package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBaseConfigWithDefaults(t *testing.T) {
	config := NewBaseConfigWithDefaults()

	// Test basic defaults
	assert.Equal(t, "ubuntu:22.04", config.Image)
	assert.Equal(t, 1000, config.UID)
	assert.Equal(t, "/opt/venv/bin", config.PythonBinPath)

	// Test container setup defaults
	assert.True(t, config.InstallHomebrew)
	assert.False(t, config.ClearLocalPackages)
	assert.False(t, config.ClearVSCodeCache)

	// Test resource defaults
	assert.Equal(t, 2, config.Resources.CPU)
	assert.Equal(t, "8Gi", config.Resources.Memory)
	assert.Equal(t, "20Gi", config.Resources.Storage)
	assert.Equal(t, 0, config.Resources.GPU)

	// Test empty slice defaults
	assert.Equal(t, []string{}, config.Packages.Python)
	assert.Equal(t, []string{}, config.Packages.APT)
	assert.Equal(t, []VolumeMount{}, config.Volumes)
}

func TestBaseConfig_GetSSHKeys(t *testing.T) {
	tests := []struct {
		name        string
		sshKeyField any
		expected    []string
		expectError bool
	}{
		{
			name:        "single string key",
			sshKeyField: "ssh-rsa AAAAB3NzaC1yc2E... user@host",
			expected:    []string{"ssh-rsa AAAAB3NzaC1yc2E... user@host"},
			expectError: false,
		},
		{
			name:        "multiple string keys",
			sshKeyField: []string{"ssh-rsa AAAAB3... user1", "ssh-ed25519 AAAAC3... user2"},
			expected:    []string{"ssh-rsa AAAAB3... user1", "ssh-ed25519 AAAAC3... user2"},
			expectError: false,
		},
		{
			name:        "interface slice from YAML",
			sshKeyField: []interface{}{"ssh-rsa AAAAB3... user1", "ssh-ed25519 AAAAC3... user2"},
			expected:    []string{"ssh-rsa AAAAB3... user1", "ssh-ed25519 AAAAC3... user2"},
			expectError: false,
		},
		{
			name:        "nil field",
			sshKeyField: nil,
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "empty string",
			sshKeyField: "",
			expected:    []string{},
			expectError: true,
		},
		{
			name:        "empty array",
			sshKeyField: []string{},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid type",
			sshKeyField: 123,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BaseConfig{SSHPublicKey: tt.sshKeyField}
			result, err := config.GetSSHKeys()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDevEnvConfig_GetUserID(t *testing.T) {
	tests := []struct {
		name     string
		uid      int
		expected string
	}{
		{
			name:     "custom UID",
			uid:      2000,
			expected: "2000",
		},
		{
			name:     "zero UID should still return zero (not default)",
			uid:      0,
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DevEnvConfig{
				BaseConfig: BaseConfig{UID: tt.uid},
			}
			result := config.GetUserID()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDevEnvConfig_CPU(t *testing.T) {
	tests := []struct {
		name     string
		cpuValue any
		expected string
	}{
		{
			name:     "integer CPU",
			cpuValue: 4,
			expected: "4",
		},
		{
			name:     "string CPU",
			cpuValue: "2.5",
			expected: "2.5",
		},
		{
			name:     "float CPU",
			cpuValue: 3.5,
			expected: "4", // float64 formatted as integer
		},
		{
			name:     "nil CPU returns 0",
			cpuValue: nil,
			expected: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DevEnvConfig{
				BaseConfig: BaseConfig{
					Resources: ResourceConfig{
						CPU: tt.cpuValue,
					},
				},
			}
			result := config.CPU()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDevEnvConfig_Memory(t *testing.T) {
	tests := []struct {
		name     string
		memory   string
		expected string
	}{
		{
			name:     "custom memory",
			memory:   "16Gi",
			expected: "16Gi",
		},
		{
			name:     "empty memory returns as-is",
			memory:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DevEnvConfig{
				BaseConfig: BaseConfig{
					Resources: ResourceConfig{
						Memory: tt.memory,
					},
				},
			}
			result := config.Memory()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDevEnvConfig_GPU(t *testing.T) {
	config := &DevEnvConfig{
		BaseConfig: BaseConfig{
			Resources: ResourceConfig{
				GPU: 2,
			},
		},
	}
	assert.Equal(t, 2, config.GPU())
}

func TestDevEnvConfig_NodePort(t *testing.T) {
	config := &DevEnvConfig{
		SSHPort: 30022,
	}
	assert.Equal(t, 30022, config.NodePort())
}

func TestDevEnvConfig_VolumeMounts(t *testing.T) {
	volumes := []VolumeMount{
		{Name: "data", LocalPath: "/local/data", ContainerPath: "/data"},
		{Name: "logs", LocalPath: "/local/logs", ContainerPath: "/logs"},
	}

	config := &DevEnvConfig{
		BaseConfig: BaseConfig{Volumes: volumes},
	}

	result := config.VolumeMounts()
	assert.Equal(t, volumes, result)
}

func TestDevEnvConfig_GetSSHKeysSlice(t *testing.T) {
	t.Run("valid SSH keys", func(t *testing.T) {
		config := &DevEnvConfig{
			BaseConfig: BaseConfig{
				SSHPublicKey: []string{"ssh-rsa AAAAB3... user1", "ssh-ed25519 AAAAC3... user2"},
			},
		}
		result := config.GetSSHKeysSlice()
		expected := []string{"ssh-rsa AAAAB3... user1", "ssh-ed25519 AAAAC3... user2"}
		assert.Equal(t, expected, result)
	})

	t.Run("invalid SSH keys returns empty slice", func(t *testing.T) {
		config := &DevEnvConfig{
			BaseConfig: BaseConfig{
				SSHPublicKey: 123, // Invalid type
			},
		}
		result := config.GetSSHKeysSlice()
		assert.Equal(t, []string{}, result)
	})
}

func TestDevEnvConfig_GetSSHKeysString(t *testing.T) {
	t.Run("valid SSH keys", func(t *testing.T) {
		config := &DevEnvConfig{
			BaseConfig: BaseConfig{
				SSHPublicKey: []string{"ssh-rsa AAAAB3... user1", "ssh-ed25519 AAAAC3... user2"},
			},
		}
		result := config.GetSSHKeysString()
		expected := "ssh-rsa AAAAB3... user1\nssh-ed25519 AAAAC3... user2\n"
		assert.Equal(t, expected, result)
	})

	t.Run("no SSH keys returns empty string", func(t *testing.T) {
		config := &DevEnvConfig{
			BaseConfig: BaseConfig{
				SSHPublicKey: nil,
			},
		}
		result := config.GetSSHKeysString()
		assert.Equal(t, "", result)
	})

	t.Run("invalid SSH keys returns empty string", func(t *testing.T) {
		config := &DevEnvConfig{
			BaseConfig: BaseConfig{
				SSHPublicKey: 123, // Invalid type
			},
		}
		result := config.GetSSHKeysString()
		assert.Equal(t, "", result)
	})
}

func TestDevEnvConfig_GetDeveloperDir(t *testing.T) {
	config := &DevEnvConfig{
		DeveloperDir: "/path/to/developers/alice",
	}
	result := config.GetDeveloperDir()
	assert.Equal(t, "/path/to/developers/alice", result)
}

func TestDevEnvConfig_CPURequest(t *testing.T) {
	config := &DevEnvConfig{
		BaseConfig: BaseConfig{
			Resources: ResourceConfig{
				CPU: "4",
			},
		},
	}
	// CPURequest should be an alias for CPU
	assert.Equal(t, config.CPU(), config.CPURequest())
}

func TestDevEnvConfig_MemoryRequest(t *testing.T) {
	config := &DevEnvConfig{
		BaseConfig: BaseConfig{
			Resources: ResourceConfig{
				Memory: "16Gi",
			},
		},
	}
	// MemoryRequest should be an alias for Memory
	assert.Equal(t, config.Memory(), config.MemoryRequest())
}

func TestDevEnvConfig_Embedding(t *testing.T) {
	// Test that BaseConfig fields are promoted properly
	config := &DevEnvConfig{
		BaseConfig: BaseConfig{
			Image:              "custom:latest",
			InstallHomebrew:    false,
			ClearLocalPackages: true,
			PythonBinPath:      "/custom/python/bin",
		},
		Name: "alice",
		Git: GitConfig{
			Name:  "Alice Smith",
			Email: "alice@example.com",
		},
	}

	// Test direct access to embedded fields
	assert.Equal(t, "custom:latest", config.Image)
	assert.False(t, config.InstallHomebrew)
	assert.True(t, config.ClearLocalPackages)
	assert.Equal(t, "/custom/python/bin", config.PythonBinPath)

	// Test user-specific fields
	assert.Equal(t, "alice", config.Name)
	assert.Equal(t, "Alice Smith", config.Git.Name)
	assert.Equal(t, "alice@example.com", config.Git.Email)
}

func TestNewBaseConfigWithDefaults_AllFieldsSet(t *testing.T) {
	config := NewBaseConfigWithDefaults()

	// Ensure no fields are left at zero values that shouldn't be
	assert.NotEmpty(t, config.Image)
	assert.NotZero(t, config.UID)
	assert.NotEmpty(t, config.PythonBinPath)
	assert.NotNil(t, config.Resources.CPU)
	assert.NotEmpty(t, config.Resources.Memory)
	assert.NotEmpty(t, config.Resources.Storage)

	// These should be initialized as empty slices, not nil
	assert.NotNil(t, config.Packages.Python)
	assert.NotNil(t, config.Packages.APT)
	assert.NotNil(t, config.Volumes)
}

func TestResourceConfig_FlexibleCPU(t *testing.T) {
	// Test that ResourceConfig can handle different CPU types
	tests := []struct {
		name     string
		cpuValue any
		valid    bool
	}{
		{"string CPU", "2.5", true},
		{"int CPU", 4, true},
		{"float64 CPU", 3.5, true},
		{"nil CPU", nil, true},
		{"bool CPU", true, false}, // Would be handled by CPU() method returning "0"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &DevEnvConfig{
				BaseConfig: BaseConfig{
					Resources: ResourceConfig{
						CPU: tt.cpuValue,
					},
				},
			}

			// The CPU() method should handle all these cases gracefully
			result := config.CPU()
			assert.IsType(t, "", result) // Should always return string
			assert.NotEmpty(t, result)   // Should never be empty
		})
	}
}
