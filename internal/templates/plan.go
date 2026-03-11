package templates

import "github.com/nauticalab/devenv-engine/internal/config"

var devBaseTemplates = []string{"statefulset", "service", "env-vars", "startup-scripts"}

var devOptionalTemplates = []string{"ingress"}

var devManagedTemplates = append(append([]string{}, devBaseTemplates...), devOptionalTemplates...)

var systemBaseTemplates = []string{"namespace"}

var systemManagedTemplates = append([]string{}, systemBaseTemplates...)

// RenderPlan defines template selection and ownership.
type RenderPlan struct {
	TemplateNames    []string
	ManagedTemplates []string
}

// BuildDevRenderPlan computes the template set from config before rendering.
func BuildDevRenderPlan(cfg *config.DevEnvConfig) RenderPlan {
	templateNames := append([]string{}, devBaseTemplates...)

	if cfg != nil && cfg.ShouldRenderIngress() {
		templateNames = append(templateNames, "ingress")
	}

	return RenderPlan{
		TemplateNames:    templateNames,
		ManagedTemplates: append([]string{}, devManagedTemplates...),
	}
}

// BuildSystemRenderPlan computes the template set for system-level manifests.
func BuildSystemRenderPlan() RenderPlan {
	return RenderPlan{
		TemplateNames:    append([]string{}, systemBaseTemplates...),
		ManagedTemplates: append([]string{}, systemManagedTemplates...),
	}
}
