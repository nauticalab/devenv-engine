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

		plan, err := BuildDevRenderPlan(cfg)
		require.NoError(t, err)

		assert.Equal(t, expectedDevTemplateNames(false), plan.TemplateNames)
	})

	t.Run("includes ingress when HTTP port is set", func(t *testing.T) {
		cfg := &config.DevEnvConfig{HTTPPort: 8080, BaseConfig: config.BaseConfig{HostName: "devenv.example.com"}}

		plan, err := BuildDevRenderPlan(cfg)
		require.NoError(t, err)

		assert.Equal(t, expectedDevTemplateNames(true), plan.TemplateNames)
	})

	t.Run("excludes ingress when hostName is missing", func(t *testing.T) {
		cfg := &config.DevEnvConfig{HTTPPort: 8080}

		plan, err := BuildDevRenderPlan(cfg)
		require.NoError(t, err)

		assert.Equal(t, expectedDevTemplateNames(false), plan.TemplateNames)
	})
}

func TestBuildDevRenderPlan_NilConfigReturnsError(t *testing.T) {
	_, err := BuildDevRenderPlan(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BuildDevRenderPlan requires non-nil config")
}

func TestBuildDevRenderPlan_Contract(t *testing.T) {
	t.Run("http disabled", func(t *testing.T) {
		plan, err := BuildDevRenderPlan(&config.DevEnvConfig{})
		require.NoError(t, err)
		assert.Equal(t, []string{"statefulset", "service", "env-vars", "startup-scripts"}, plan.TemplateNames)
	})

	t.Run("http enabled", func(t *testing.T) {
		plan, err := BuildDevRenderPlan(&config.DevEnvConfig{HTTPPort: 8080, BaseConfig: config.BaseConfig{HostName: "devenv.example.com"}})
		require.NoError(t, err)
		assert.Equal(t, []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"}, plan.TemplateNames)
	})
}

func TestBuildSystemRenderPlan(t *testing.T) {
	plan := BuildSystemRenderPlan()

	assert.Equal(t, copyTemplateNames(systemTemplates), plan.TemplateNames)
}

func TestBuildSystemRenderPlan_Contract(t *testing.T) {
	plan := BuildSystemRenderPlan()
	assert.Equal(t, []string{"namespace"}, plan.TemplateNames)
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
