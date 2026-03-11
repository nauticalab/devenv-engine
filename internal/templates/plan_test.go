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

		plan := BuildDevRenderPlan(cfg)

		assert.Equal(t, expectedDevTemplateNames(false), plan.TemplateNames)
		assert.Equal(t, copyTemplateNames(devManagedTemplates), plan.ManagedTemplates)
	})

	t.Run("includes ingress when HTTP port is set", func(t *testing.T) {
		cfg := &config.DevEnvConfig{HTTPPort: 8080, BaseConfig: config.BaseConfig{HostName: "devenv.example.com"}}

		plan := BuildDevRenderPlan(cfg)

		assert.Equal(t, expectedDevTemplateNames(true), plan.TemplateNames)
		assert.Equal(t, copyTemplateNames(devManagedTemplates), plan.ManagedTemplates)
	})

	t.Run("excludes ingress when hostName is missing", func(t *testing.T) {
		cfg := &config.DevEnvConfig{HTTPPort: 8080}

		plan := BuildDevRenderPlan(cfg)

		assert.Equal(t, expectedDevTemplateNames(false), plan.TemplateNames)
		assert.Equal(t, copyTemplateNames(devManagedTemplates), plan.ManagedTemplates)
	})
}

func TestBuildDevRenderPlan_Contract(t *testing.T) {
	t.Run("http disabled", func(t *testing.T) {
		plan := BuildDevRenderPlan(&config.DevEnvConfig{})
		assert.Equal(t, []string{"statefulset", "service", "env-vars", "startup-scripts"}, plan.TemplateNames)
		assert.Equal(t, []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"}, plan.ManagedTemplates)
	})

	t.Run("http enabled", func(t *testing.T) {
		plan := BuildDevRenderPlan(&config.DevEnvConfig{HTTPPort: 8080, BaseConfig: config.BaseConfig{HostName: "devenv.example.com"}})
		assert.Equal(t, []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"}, plan.TemplateNames)
		assert.Equal(t, []string{"statefulset", "service", "env-vars", "startup-scripts", "ingress"}, plan.ManagedTemplates)
	})
}

func TestBuildSystemRenderPlan(t *testing.T) {
	plan := BuildSystemRenderPlan()

	assert.Equal(t, copyTemplateNames(systemBaseTemplates), plan.TemplateNames)
	assert.Equal(t, copyTemplateNames(systemManagedTemplates), plan.ManagedTemplates)
}

func TestBuildSystemRenderPlan_Contract(t *testing.T) {
	plan := BuildSystemRenderPlan()
	assert.Equal(t, []string{"namespace"}, plan.TemplateNames)
	assert.Equal(t, []string{"namespace"}, plan.ManagedTemplates)
}

func TestRenderPlans_TargetTemplatesAreManaged(t *testing.T) {
	t.Run("dev plan", func(t *testing.T) {
		plan := BuildDevRenderPlan(&config.DevEnvConfig{HTTPPort: 8080, BaseConfig: config.BaseConfig{HostName: "devenv.example.com"}})
		requireTargetSubsetOfManaged(t, plan.TemplateNames, plan.ManagedTemplates)
	})

	t.Run("system plan", func(t *testing.T) {
		plan := BuildSystemRenderPlan()
		requireTargetSubsetOfManaged(t, plan.TemplateNames, plan.ManagedTemplates)
	})
}

func requireTargetSubsetOfManaged(t *testing.T, targetTemplates []string, managedTemplates []string) {
	t.Helper()

	managedSet := make(map[string]struct{}, len(managedTemplates))
	for _, templateName := range managedTemplates {
		managedSet[templateName] = struct{}{}
	}

	for _, templateName := range targetTemplates {
		_, ok := managedSet[templateName]
		require.Truef(t, ok, "target template %q is not managed", templateName)
	}
}

func expectedDevTemplateNames(includeOptional bool) []string {
	templateNames := copyTemplateNames(devBaseTemplates)
	if includeOptional {
		templateNames = append(templateNames, devOptionalTemplates...)
	}
	return templateNames
}

func copyTemplateNames(templateNames []string) []string {
	return append([]string{}, templateNames...)
}
