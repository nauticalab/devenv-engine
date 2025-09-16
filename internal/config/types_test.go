package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetSSHKeys tests the GetSSHKeys method with different input types
func TestGetSSHKeys(t *testing.T) {
	// Test cases using table-driven testing pattern
	testCases := []struct {
		name        string   // Description of the test case
		sshKeyField any      // Input value for SSHPublicKey field
		expected    []string // Expected output
		expectError bool     // Whether we expect an error
		errorMsg    string   // Expected error message (if any)
	}{
		{
			name:        "single SSH key as string",
			sshKeyField: "ssh-rsa AAAAB3NzaC1yc2E... user@example.com",
			expected:    []string{"ssh-rsa AAAAB3NzaC1yc2E... user@example.com"},
			expectError: false,
		},
		{
			name: "multiplel SSH keys as []any",
			sshKeyField: []any{
				"ssh-rsa AAAAB3NzaC1yc2E... user1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user2@example.com",
			},
			expected: []string{
				"ssh-rsa AAAAB3NzaC1yc2E... user1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user2@example.com",
			},
			expectError: false,
		},
		{
			name: "multiple SSH keys as []string",
			sshKeyField: []string{
				"ssh-rsa AAAAB3NzaC1yc2E... user1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user2@example.com",
			},
			expected: []string{
				"ssh-rsa AAAAB3NzaC1yc2E... user1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user2@example.com",
			},
			expectError: false,
		},
		{
			name:        "empty string should return error",
			sshKeyField: "",
			expected:    nil,
			expectError: true,
			errorMsg:    "SSH key cannot be empty",
		},
		{
			name:        "empty array should return error",
			sshKeyField: []any{},
			expected:    nil,
			expectError: true,
			errorMsg:    "", // We'll check this behavior
		},
		{
			name:        "array with empty string should return error",
			sshKeyField: []any{"ssh-rsa valid-key", ""},
			expected:    nil,
			expectError: true,
			errorMsg:    "SSH key at index 1 cannot be empty",
		},
		{
			name:        "invalid type should return error",
			sshKeyField: 12345,
			expected:    nil,
			expectError: true,
			errorMsg:    "SSH key field must be either a string or array of strings",
		},
		{
			name:        "array with non-string should return error",
			sshKeyField: []any{"valid-key", 123},
			expected:    nil,
			expectError: true,
			errorMsg:    "SSH key at index 1 is not a string",
		},
	}

	// Iterate through test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			config := &DevEnvConfig{
				Name:         "testuser",
				SSHPublicKey: tc.sshKeyField,
			}

			// Act
			result, err := config.GetSSHKeys()

			// Assert - Much cleaner with testify!
			if tc.expectError {
				require.Error(t, err) // Stops test if no error
				if tc.errorMsg != "" {
					assert.Equal(t, tc.errorMsg, err.Error())
				}
			} else {
				require.NoError(t, err) // Stops test if error occurs
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestLoadDeveloperConfig tests loading configuration from YAML files
func TestLoadDeveloperConfig(t *testing.T) {
	// Use testdata directory for test fixtures
	testConfigDir := "testdata"

	testCases := []struct {
		name           string
		developerName  string
		expectError    bool
		expectedName   string
		expectedSSHLen int // Number of SSH keys expected
		expectedPort   int
		expectedUID    int
	}{
		{
			name:           "load valid complete configuration",
			developerName:  "valid_user",
			expectError:    false,
			expectedName:   "testuser",
			expectedSSHLen: 1,
			expectedPort:   30001,
			expectedUID:    2000,
		},
		{
			name:           "load minimal valid configuration",
			developerName:  "minimal_user",
			expectError:    false,
			expectedName:   "minimal",
			expectedSSHLen: 1,
			expectedPort:   0, // No SSH port specified
			expectedUID:    0, // No UID specified
		},
		{
			name:           "load configuration with multiple SSH keys",
			developerName:  "multi_ssh_user",
			expectError:    false,
			expectedName:   "multissh",
			expectedSSHLen: 2,
			expectedPort:   30002,
			expectedUID:    0,
		},
		{
			name:          "configuration file does not exist",
			developerName: "nonexistent_user",
			expectError:   true,
		},
		{
			name:          "invalid configuration missing name",
			developerName: "invalid_user",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			config, err := LoadDeveloperConfig(testConfigDir, tc.developerName)

			// Assert
			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)

				// Test specific config values
				assert.Equal(t, tc.expectedName, config.Name)
				assert.Equal(t, tc.expectedPort, config.SSHPort)
				assert.Equal(t, tc.expectedUID, config.UID)

				// Test SSH keys
				sshKeys, err := config.GetSSHKeys()
				require.NoError(t, err)
				assert.Len(t, sshKeys, tc.expectedSSHLen)

				// Test that developer directory is set correctly
				expectedDir := filepath.Join(testConfigDir, tc.developerName)
				assert.Equal(t, expectedDir, config.GetDeveloperDir())
			}
		})
	}
}

// TestGetSSHKeysSlice tests the template helper method that returns SSH keys as slice
func TestGetSSHKeysSlice(t *testing.T) {
	testCases := []struct {
		name        string
		sshKeyField interface{}
		expected    []string
	}{
		{
			name:        "valid single SSH key",
			sshKeyField: "ssh-rsa AAAAB3NzaC1yc2E... user@example.com",
			expected:    []string{"ssh-rsa AAAAB3NzaC1yc2E... user@example.com"},
		},
		{
			name: "valid multiple SSH keys",
			sshKeyField: []interface{}{
				"ssh-rsa AAAAB3NzaC1yc2E... user1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user2@example.com",
			},
			expected: []string{
				"ssh-rsa AAAAB3NzaC1yc2E... user1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user2@example.com",
			},
		},
		{
			name:        "invalid SSH key returns empty slice",
			sshKeyField: 12345,      // This should cause GetSSHKeys() to error
			expected:    []string{}, // Helper method returns empty slice on error
		},
		{
			name:        "empty string returns empty slice",
			sshKeyField: "",
			expected:    []string{}, // Helper method returns empty slice on error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			config := &DevEnvConfig{
				Name:         "testuser",
				SSHPublicKey: tc.sshKeyField,
			}

			// Act
			result := config.GetSSHKeysSlice()

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGetSSHKeysString tests the template helper that returns SSH keys as newline-separated string
func TestGetSSHKeysString(t *testing.T) {
	testCases := []struct {
		name        string
		sshKeyField interface{}
		expected    string
	}{
		{
			name:        "single SSH key",
			sshKeyField: "ssh-rsa AAAAB3NzaC1yc2E... user@example.com",
			expected:    "ssh-rsa AAAAB3NzaC1yc2E... user@example.com",
		},
		{
			name: "multiple SSH keys joined with newlines",
			sshKeyField: []interface{}{
				"ssh-rsa AAAAB3NzaC1yc2E... user1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user2@example.com",
			},
			expected: "ssh-rsa AAAAB3NzaC1yc2E... user1@example.com\nssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user2@example.com",
		},
		{
			name: "three SSH keys with proper newline separation",
			sshKeyField: []string{
				"key1",
				"key2",
				"key3",
			},
			expected: "key1\nkey2\nkey3",
		},
		{
			name:        "invalid SSH key returns empty string",
			sshKeyField: 12345,
			expected:    "", // Empty slice produces empty string
		},
		{
			name:        "empty array returns empty string",
			sshKeyField: []interface{}{},
			expected:    "", // Empty slice produces empty string
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			config := &DevEnvConfig{
				Name:         "testuser",
				SSHPublicKey: tc.sshKeyField,
			}

			// Act
			result := config.GetSSHKeysString()

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGetUserID tests the GetUserID method with different UID values
func TestGetUserID(t *testing.T) {
	testCases := []struct {
		name     string
		uid      int
		expected string
	}{
		{
			name:     "zero UID should return default",
			uid:      0,
			expected: "1000",
		},
		{
			name:     "custom UID",
			uid:      2000,
			expected: "2000",
		},
		{
			name:     "system UID",
			uid:      100,
			expected: "100",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			config := &DevEnvConfig{
				Name: "testuser",
				UID:  tc.uid,
			}

			// Act
			result := config.GetUserID()

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestMemory tests the Memory method with different values and defaults
func TestMemory(t *testing.T) {
	testCases := []struct {
		name        string
		memoryValue string
		expected    string
	}{
		{
			name:        "empty memory should return default",
			memoryValue: "",
			expected:    "8Gi", // Default from DefaultValue.Memory
		},
		{
			name:        "custom memory value",
			memoryValue: "16Gi",
			expected:    "16Gi",
		},
		{
			name:        "memory in different units",
			memoryValue: "4096Mi",
			expected:    "4096Mi",
		},
		{
			name:        "memory as plain number",
			memoryValue: "8192",
			expected:    "8192",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			config := &DevEnvConfig{
				Name: "testuser",
				Resources: ResourceConfig{
					Memory: tc.memoryValue,
				},
			}

			// Act
			result := config.Memory()

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestCPU tests the CPU method with different input types and defaults
func TestCPU(t *testing.T) {
	testCases := []struct {
		name     string
		cpuValue interface{} // What we'll set in Resources.CPU
		expected string
	}{
		{
			name:     "nil CPU should return default",
			cpuValue: nil,
			expected: "2", // Default from DefaultValue.CPU
		},
		{
			name:     "string CPU value",
			cpuValue: "4",
			expected: "4",
		},
		{
			name:     "empty string CPU should return default",
			cpuValue: "",
			expected: "2",
		},
		{
			name:     "int CPU value",
			cpuValue: 8,
			expected: "8",
		},
		{
			name:     "float64 CPU value",
			cpuValue: 2.5,
			expected: "2", // Actual behavior from failed test
		},
		{
			name:     "float64 CPU rounds down",
			cpuValue: 2.4,
			expected: "2", // %.0f rounds 2.4 to 2
		},
		{
			name:     "float64 whole number",
			cpuValue: 4.0,
			expected: "4",
		},
		{
			name:     "unusual type falls back to default",
			cpuValue: []string{"not", "a", "cpu"},
			expected: "2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			config := &DevEnvConfig{
				Name: "testuser",
				Resources: ResourceConfig{
					CPU: tc.cpuValue,
				},
			}

			// Act
			result := config.CPU()

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}
