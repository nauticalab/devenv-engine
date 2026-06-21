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
			UID:       2000,
			Image:     "ubuntu:22.04",
			Namespace: "devenv-test",
			HostName:  "devenv.example.com",
			HomeDirMountBase: "/mnt/devenv",
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

	templates := []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"}

	for _, templateName := range templates {
		t.Run(templateName, func(t *testing.T) {
			// Create temporary output directory
			tempDir := t.TempDir()

			// Create renderer
			plan, err := BuildDevRenderPlan(testConfig)
			require.NoError(t, err)
			renderer := NewDevRenderer(tempDir, testConfig, plan.TemplateNames)

			// Render template
			err = renderer.RenderTemplate(templateName)
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
	t.Run("includes ingress when HTTP port is set", func(t *testing.T) {
		testConfig := &config.DevEnvConfig{
			Name: "minimal",
			BaseConfig: config.BaseConfig{
				SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... minimal@example.com",
				Namespace:    "devenv-test",
				HostName:     "devenv.example.com",
			},
			SSHPort:  30002,
			HTTPPort: 8080,
		}

		tempDir := t.TempDir()
		plan, err := BuildDevRenderPlan(testConfig)
		require.NoError(t, err)
		renderer := NewDevRenderer(tempDir, testConfig, plan.TemplateNames)

		err = renderer.RenderAll()
		require.NoError(t, err, "RenderAll should not return error")

		expectedFiles := templateNamesToFiles(plan.TemplateNames)

		for _, filename := range expectedFiles {
			filePath := filepath.Join(tempDir, filename)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "Expected file %s should exist", filename)

			content, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.NotEmpty(t, content, "File %s should not be empty", filename)
		}
	})

	t.Run("skips ingress when HTTP port is unset", func(t *testing.T) {
		testConfig := &config.DevEnvConfig{
			Name: "minimal",
			BaseConfig: config.BaseConfig{
				SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... minimal@example.com",
				Namespace:    "devenv-test",
			},
			SSHPort: 30002,
		}

		tempDir := t.TempDir()
		plan, err := BuildDevRenderPlan(testConfig)
		require.NoError(t, err)
		renderer := NewDevRenderer(tempDir, testConfig, plan.TemplateNames)

		err = renderer.RenderAll()
		require.NoError(t, err, "RenderAll should not return error")

		expectedFiles := templateNamesToFiles(plan.TemplateNames)
		for _, filename := range expectedFiles {
			filePath := filepath.Join(tempDir, filename)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "Expected file %s should exist", filename)
		}

		_, err = os.Stat(filepath.Join(tempDir, "ingress.yaml"))
		assert.ErrorIs(t, err, os.ErrNotExist, "ingress.yaml should not be generated without HTTP port")
	})

	t.Run("preserves stale ingress when render fails", func(t *testing.T) {
		testConfig := &config.DevEnvConfig{
			Name: "minimal",
			BaseConfig: config.BaseConfig{
				SSHPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... minimal@example.com",
				Namespace:    "devenv-test",
			},
			SSHPort: 30002,
		}

		tempDir := t.TempDir()
		staleIngress := filepath.Join(tempDir, "ingress.yaml")
		require.NoError(t, os.WriteFile(staleIngress, []byte("stale"), 0o644))

		renderer := NewRenderer[config.DevEnvConfig](
			tempDir,
			"template_files/dev",
			[]string{"nonexistent-template"},
			testConfig,
		)

		err := renderer.RenderAll()
		require.Error(t, err)

		content, readErr := os.ReadFile(staleIngress)
		require.NoError(t, readErr)
		assert.Equal(t, "stale", string(content), "stale ingress.yaml should be preserved when render fails")
	})
}

func templateNamesToFiles(templateNames []string) []string {
	files := make([]string, 0, len(templateNames))
	for _, templateName := range templateNames {
		files = append(files, templateName+".yaml")
	}
	return files
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
		plan, err := BuildDevRenderPlan(testConfig)
		require.NoError(t, err)
		renderer := NewDevRenderer(tempDir, testConfig, plan.TemplateNames)

		err = renderer.RenderTemplate("nonexistent")
		assert.Error(t, err, "Should return error for invalid template")
	})

	t.Run("invalid output directory", func(t *testing.T) {
		// Make the parent path a file so MkdirAll fails deterministically.
		parentFile := filepath.Join(t.TempDir(), "not-a-directory")
		require.NoError(t, os.WriteFile(parentFile, []byte("x"), 0o644))
		plan, err := BuildDevRenderPlan(testConfig)
		require.NoError(t, err)
		renderer := NewDevRenderer(filepath.Join(parentFile, "child"), testConfig, plan.TemplateNames)

		err = renderer.RenderTemplate("env-vars")
		assert.Error(t, err, "Should return error for invalid output directory")
	})
}

// Command-line flag for updating golden files
// Usage: go test -v ./internal/templates -update-golden
var updateGolden = flag.Bool("update-golden", false, "update golden files")
