package templates

import (
	"testing"

	"github.com/nauticalab/devenv-engine/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDevRenderPlan(t *testing.T) {
	t.Run("excludes ingress when HTTP port is unset", func(t *testing.T) {
		cfg := &config.DevEnvConfig{}

		templateNames, err := BuildDevRenderPlan(cfg)
		require.NoError(t, err)

		assert.Equal(t, expectedDevTemplateNames(false), templateNames)
	})

	t.Run("includes ingress when HTTP port is set", func(t *testing.T) {
		cfg := &config.DevEnvConfig{HTTPPort: 8080, BaseConfig: config.BaseConfig{HostName: "devenv.example.com"}}

		templateNames, err := BuildDevRenderPlan(cfg)
		require.NoError(t, err)

		assert.Equal(t, expectedDevTemplateNames(true), templateNames)
	})

	t.Run("excludes ingress when hostName is missing", func(t *testing.T) {
		cfg := &config.DevEnvConfig{HTTPPort: 8080}

		templateNames, err := BuildDevRenderPlan(cfg)
		require.NoError(t, err)

		assert.Equal(t, expectedDevTemplateNames(false), templateNames)
	})
}

func TestBuildDevRenderPlan_Contract(t *testing.T) {
	t.Run("http disabled", func(t *testing.T) {
		templateNames, err := BuildDevRenderPlan(&config.DevEnvConfig{})
		require.NoError(t, err)
		assert.Equal(t, []string{"statefulset", "service", "env-vars", "startup-scripts"}, templateNames)
	})

	t.Run("http enabled", func(t *testing.T) {
		templateNames, err := BuildDevRenderPlan(&config.DevEnvConfig{HTTPPort: 8080, BaseConfig: config.BaseConfig{HostName: "devenv.example.com"}})
		require.NoError(t, err)
		assert.Equal(t, []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"}, templateNames)
	})
}

func TestBuildSystemRenderPlan(t *testing.T) {
	templateNames := BuildSystemRenderPlan()

	assert.Equal(t, copyTemplateNames(systemTemplates), templateNames)
}

func TestBuildSystemRenderPlan_Contract(t *testing.T) {
	templateNames := BuildSystemRenderPlan()
	assert.Equal(t, []string{"namespace"}, templateNames)
}

func TestTemplateScopes(t *testing.T) {
	assert.Equal(t, []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"}, DevCleanupScope())
	assert.Equal(t, []string{"namespace"}, SystemCleanupScope())
}

func expectedDevTemplateNames(includeOptional bool) []string {
	templateNames := []string{"statefulset", "service", "env-vars", "startup-scripts"}
	if includeOptional {
		templateNames = append(templateNames, "ingress")
	}
	return templateNames
}

func copyTemplateNames(templateNames []string) []string {
	return append([]string{}, templateNames...)
}
