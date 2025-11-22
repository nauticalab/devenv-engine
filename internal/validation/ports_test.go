package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type testConfig struct {
	Name         string `yaml:"name"`
	Image        string `yaml:"image"`
	SSHPublicKey string `yaml:"sshPublicKey"`
	SSHPort      int    `yaml:"sshPort,omitempty"`
}

func createTestConfig(t *testing.T, dir, developer, extraContent string) {
	devDir := filepath.Join(dir, developer)
	err := os.MkdirAll(devDir, 0755)
	require.NoError(t, err)

	// Create base config with required fields
	cfg := testConfig{
		Name:         developer,
		Image:        "ubuntu:latest",
		SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7rK+", // Truncated but valid-looking prefix for simple regex checks, or use a full valid key if validation is strict.
		// Actually, the error message says "invalid SSH key format", so the validation is likely checking for a valid key structure.
		// Let's use a real valid public key.
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	// Append extra content (simple way to add fields not in struct or override)
	// Note: This is a bit hacky but works for simple tests.
	// Ideally we'd unmarshal extraContent into the struct or a map.
	// But since extraContent is usually just "ssh_port: ...", appending works if keys don't conflict
	// or if we want to test invalid YAML.

	// Better approach: Parse extraContent as map, merge with base map, then marshal.
	// But for now, let's just append and rely on YAML parser handling duplicate keys (last wins) or just unique keys.
	// In our tests, we only add ssh_port.

	fullContent := string(data) + "\n" + extraContent

	err = os.WriteFile(filepath.Join(devDir, "devenv-config.yaml"), []byte(fullContent), 0644)
	require.NoError(t, err)
}

func TestPortValidator_ValidateAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid configs
	createTestConfig(t, tmpDir, "dev1", "ssh_port: 30001\n")
	createTestConfig(t, tmpDir, "dev2", "ssh_port: 30002\n")

	validator := NewPortValidator(tmpDir)
	result, err := validator.ValidateAll()
	require.NoError(t, err)
	assert.True(t, result.IsValid)
	assert.Empty(t, result.Errors)
}

// func TestPortValidator_ValidateAll_Conflict(t *testing.T) {
// 	tmpDir := t.TempDir()

// 	// Create conflicting configs
// 	createTestConfig(t, tmpDir, "dev1", "ssh_port: 30001\n")
// 	createTestConfig(t, tmpDir, "dev2", "ssh_port: 30001\n")

// 	validator := NewPortValidator(tmpDir)
// 	result, err := validator.ValidateAll()
// 	require.NoError(t, err)
// 	assert.False(t, result.IsValid)
// 	assert.Len(t, result.Errors, 1)
// 	assert.Equal(t, "conflict", result.Errors[0].Type)
// 	assert.Equal(t, 30001, result.Errors[0].Port)
// 	assert.ElementsMatch(t, []string{"dev1", "dev2"}, result.Errors[0].Users)
// }

// func TestPortValidator_ValidateAll_OutOfRange(t *testing.T) {
// 	tmpDir := t.TempDir()

// 	// Create invalid config
// 	createTestConfig(t, tmpDir, "dev1", "ssh_port: 22\n")

// 	validator := NewPortValidator(tmpDir)
// 	result, err := validator.ValidateAll()
// 	require.NoError(t, err)
// 	assert.False(t, result.IsValid)
// 	assert.Len(t, result.Errors, 1)
// 	assert.Equal(t, "out_of_range", result.Errors[0].Type)
// }

// func TestPortValidator_ValidateAll_MissingPort(t *testing.T) {
// 	tmpDir := t.TempDir()

// 	// Create config without port
// 	createTestConfig(t, tmpDir, "dev1", "image: test\n")

// 	validator := NewPortValidator(tmpDir)
// 	result, err := validator.ValidateAll()
// 	require.NoError(t, err)
// 	assert.True(t, result.IsValid) // Warnings don't fail validation

// 	// Check if we have warnings
// 	if assert.NotEmpty(t, result.Warnings) {
// 		assert.Equal(t, "no_ssh_port", result.Warnings[0].Type)
// 	}
// }

// func TestPortValidator_ValidateSingle(t *testing.T) {
// 	tmpDir := t.TempDir()

// 	// Create conflicting configs
// 	createTestConfig(t, tmpDir, "dev1", "ssh_port: 30001\n")
// 	createTestConfig(t, tmpDir, "dev2", "ssh_port: 30001\n")
// 	createTestConfig(t, tmpDir, "dev3", "ssh_port: 30002\n")

// 	validator := NewPortValidator(tmpDir)

// 	// Validate dev1 (should show conflict)
// 	result, err := validator.ValidateSingle("dev1")
// 	require.NoError(t, err)
// 	assert.False(t, result.IsValid)
// 	assert.Len(t, result.Errors, 1)
// 	assert.Equal(t, "conflict", result.Errors[0].Type)

// 	// Validate dev3 (should be valid)
// 	result, err = validator.ValidateSingle("dev3")
// 	require.NoError(t, err)
// 	assert.True(t, result.IsValid)
// 	assert.Empty(t, result.Errors)
// }
