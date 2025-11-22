package templates

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/nauticalab/devenv-engine/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderTemplate tests individual template rendering with golden files
func TestRenderTemplate(t *testing.T) {
	// Create test configuration
	testConfig := &config.DevEnvConfig{
		Name: "testuser",

		SSHPort:  30001,
		HTTPPort: 8080,
		BaseConfig: config.BaseConfig{
			SSHPublicKey: []any{
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... testuser@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI... testuser2@example.com",
			},
			UID:   2000,
			Image: "ubuntu:22.04",
			Packages: config.PackageConfig{
				Python: []string{"numpy", "pandas"},
				APT:    []string{"vim", "curl"},
			},
			Resources: config.ResourceConfig{
				CPU:     "4",
				Memory:  "16Gi",
				Storage: "100Gi",
				GPU:     2,
			},
			Volumes: []config.VolumeMount{
				{
					Name:          "data-volume",
					LocalPath:     "/mnt/data",
					ContainerPath: "/data",
				},
				{
					Name:          "config-volume",
					LocalPath:     "/mnt/config",
					ContainerPath: "/config",
				},
			},
		},
		IsAdmin:     true,
		TargetNodes: []string{"node1", "node2"},
		Git: config.GitConfig{
			Name:  "Test User",
			Email: "testuser@example.com",
		},
	}

	templates := []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress", "serviceaccount"}

	for _, templateName := range templates {
		t.Run(templateName, func(t *testing.T) {
			// Create temporary output directory
			tempDir := t.TempDir()

			// Create renderer
			renderer := NewDevRenderer(tempDir)

			// Render template
			err := renderer.RenderTemplate(templateName, testConfig)
			require.NoError(t, err, "Failed to render template %s", templateName)

			// Read the generated output
			outputPath := filepath.Join(tempDir, templateName+".yaml")
			actualOutput, err := os.ReadFile(outputPath)
			require.NoError(t, err, "Failed to read rendered output")

			// Compare with golden file
			goldenPath := filepath.Join("testdata", "golden", templateName+".yaml")

			if *updateGolden {
				// Update mode: write actual output to golden file
				err := os.MkdirAll(filepath.Dir(goldenPath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(goldenPath, actualOutput, 0644)
				require.NoError(t, err)
				t.Logf("Updated golden file: %s", goldenPath)
				return // Skip comparison in update mode
			}

			// Test mode: compare against golden file
			expectedOutput, err := os.ReadFile(goldenPath)
			if os.IsNotExist(err) {
				t.Fatalf("Golden file does not exist: %s. Run with UPDATE_GOLDEN=1 to create it.", goldenPath)
			}
			require.NoError(t, err, "Failed to read golden file %s", goldenPath)

			assert.Equal(t, string(expectedOutput), string(actualOutput),
				"Template output doesn't match golden file for %s", templateName)
		})
	}
}

// TestRenderAll tests the RenderAll function that renders all templates
func TestRenderAll(t *testing.T) {
	// Create minimal test configuration
	testConfig := &config.DevEnvConfig{
		Name: "minimal",
		BaseConfig: config.BaseConfig{
			SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... minimal@example.com",
		},
		SSHPort: 30002,
	}

	tempDir := t.TempDir()
	renderer := NewDevRenderer(tempDir)

	// Test RenderAll
	err := renderer.RenderAll(testConfig)
	require.NoError(t, err, "RenderAll should not return error")

	// Verify all expected files were created
	expectedFiles := []string{"statefulset.yaml", "service.yaml", "env-vars.yaml", "startup-scripts.yaml", "ingress.yaml", "serviceaccount.yaml"}

	for _, filename := range expectedFiles {
		filePath := filepath.Join(tempDir, filename)
		_, err := os.Stat(filePath)
		assert.NoError(t, err, "Expected file %s should exist", filename)

		// Verify file is not empty
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.NotEmpty(t, content, "File %s should not be empty", filename)
	}
}

// TestRenderTemplate_ErrorCases tests error handling in template rendering
func TestRenderTemplate_ErrorCases(t *testing.T) {
	testConfig := &config.DevEnvConfig{
		Name: "testuser",
		BaseConfig: config.BaseConfig{
			SSHPublicKey: "ssh-rsa AAAAB3... testuser@example.com",
		},
	}

	t.Run("invalid template name", func(t *testing.T) {
		tempDir := t.TempDir()
		renderer := NewDevRenderer(tempDir)

		err := renderer.RenderTemplate("non_existent_template", &config.DevEnvConfig{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file does not exist")
	})

	t.Run("invalid output directory", func(t *testing.T) {
		// Use a path that can't be created (assuming /root is not writable in test)
		renderer := NewDevRenderer("/invalid/path/that/cannot/be/created")

		err := renderer.RenderTemplate("statefulset", testConfig)
		assert.Error(t, err, "Should return error for invalid output directory")
	})
}

func TestTemplateFuncs(t *testing.T) {
	funcs := templateFuncs("template_files/dev")

	// Test b64enc
	b64enc := funcs["b64enc"].(func(string) string)
	assert.Equal(t, "aGVsbG8=", b64enc("hello"))

	// Test indent
	indent := funcs["indent"].(func(int, string) string)
	assert.Equal(t, "line1\n  line2", indent(2, "line1\nline2"))

	// Test getStaticScript (mocking FS is hard here, but we can try reading existing one)
	// We need to ensure the path exists relative to where test runs.
	// The renderer uses embedded FS, so we can't easily mock it without changing the code to accept FS interface.
	// But we can test that the function exists and returns error for missing file.
	getStaticScript := funcs["getStaticScript"].(func(string) (string, error))
	_, err := getStaticScript("non-existent-script")
	assert.Error(t, err)

	// Test getTemplatedScript
	getTemplatedScript := funcs["getTemplatedScript"].(func(string, *config.DevEnvConfig) (string, error))
	_, err = getTemplatedScript("non-existent-script", &config.DevEnvConfig{})
	assert.Error(t, err)
}

// Command-line flag for updating golden files
// Usage: go test -v ./internal/templates -update-golden
var updateGolden = flag.Bool("update-golden", false, "update golden files")
