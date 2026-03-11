package templates

import "github.com/nauticalab/devenv-engine/internal/config"

type GenerationSpec[T config.BaseConfig | config.DevEnvConfig] struct {
	Config            *T
	Plan              RenderPlan
	OutputDir         string
	TemplateRoot      string
	PostRenderOptions PostRenderOptions
}

func BuildDevGenerationSpec(cfg *config.DevEnvConfig, outputDir string, postRenderOpts PostRenderOptions) GenerationSpec[config.DevEnvConfig] {
	return GenerationSpec[config.DevEnvConfig]{
		Config:            cfg,
		Plan:              BuildDevRenderPlan(cfg),
		OutputDir:         outputDir,
		TemplateRoot:      "template_files/dev",
		PostRenderOptions: postRenderOpts,
	}
}

func BuildSystemGenerationSpec(cfg *config.BaseConfig, outputDir string, postRenderOpts PostRenderOptions) GenerationSpec[config.BaseConfig] {
	return GenerationSpec[config.BaseConfig]{
		Config:            cfg,
		Plan:              BuildSystemRenderPlan(),
		OutputDir:         outputDir,
		TemplateRoot:      "template_files/system",
		PostRenderOptions: postRenderOpts,
	}
}
