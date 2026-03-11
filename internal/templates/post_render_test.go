package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPostRender_RemovesUnplannedManagedOutputs(t *testing.T) {
	tempDir := t.TempDir()
	staleIngress := filepath.Join(tempDir, "ingress.yaml")
	require.NoError(t, os.WriteFile(staleIngress, []byte("stale"), 0o644))

	plan := RenderPlan{
		TemplateNames:    []string{"statefulset", "service", "env-vars", "startup-scripts"},
		ManagedTemplates: []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"},
	}

	err := RunPostRender(tempDir, plan, NewPostRenderOptions(true))
	require.NoError(t, err)

	_, err = os.Stat(staleIngress)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestRunPostRender_PreservesPlannedOutputs(t *testing.T) {
	tempDir := t.TempDir()
	plannedIngress := filepath.Join(tempDir, "ingress.yaml")
	require.NoError(t, os.WriteFile(plannedIngress, []byte("planned"), 0o644))

	plan := RenderPlan{
		TemplateNames:    []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"},
		ManagedTemplates: []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"},
	}

	err := RunPostRender(tempDir, plan, NewPostRenderOptions(true))
	require.NoError(t, err)

	content, err := os.ReadFile(plannedIngress)
	require.NoError(t, err)
	assert.Equal(t, "planned", string(content))
}

func TestRunPostRender_IgnoresUnmanagedOutputs(t *testing.T) {
	tempDir := t.TempDir()
	unmanagedFile := filepath.Join(tempDir, "notes.yaml")
	require.NoError(t, os.WriteFile(unmanagedFile, []byte("keep"), 0o644))

	plan := RenderPlan{
		TemplateNames:    []string{"namespace"},
		ManagedTemplates: []string{"namespace"},
	}

	err := RunPostRender(tempDir, plan, NewPostRenderOptions(true))
	require.NoError(t, err)

	content, err := os.ReadFile(unmanagedFile)
	require.NoError(t, err)
	assert.Equal(t, "keep", string(content))
}

func TestRunPostRender_PreservesStaleOutputsWhenCleanupDisabled(t *testing.T) {
	tempDir := t.TempDir()
	staleIngress := filepath.Join(tempDir, "ingress.yaml")
	require.NoError(t, os.WriteFile(staleIngress, []byte("stale"), 0o644))

	plan := RenderPlan{
		TemplateNames:    []string{"statefulset", "service", "env-vars", "startup-scripts"},
		ManagedTemplates: []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"},
	}

	err := RunPostRender(tempDir, plan, PostRenderOptions{CleanupUnplanned: false})
	require.NoError(t, err)

	content, err := os.ReadFile(staleIngress)
	require.NoError(t, err)
	assert.Equal(t, "stale", string(content))
}
