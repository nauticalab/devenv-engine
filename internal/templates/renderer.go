package templates

import (
	"embed"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/nauticalab/devenv-engine/internal/config"
)

var devTemplatesToRender = []string{"serviceaccount", "statefulset", "service", "env-vars",
	"startup-scripts", "ingress"}

var systemTemplatesToRender = []string{"namespace", "manager/deployment", "manager/service", "manager/serviceaccount", "manager/rbac"}

// Embed all devTemplates and scripts at compile time
//
//go:embed template_files
var templates embed.FS

// Renderer handles template operations
type Renderer[T config.BaseConfig | config.DevEnvConfig] struct {
	outputDir       string
	templateRoot    string
	targetTemplates []string
}

// NewRenderer creates a new template renderer
func NewDevRenderer(outputDir string) *Renderer[config.DevEnvConfig] {
	return NewRendererWithFS[config.DevEnvConfig](outputDir, "template_files/dev", devTemplatesToRender)
}

func NewSystemRenderer(outputDir string) *Renderer[config.BaseConfig] {
	return NewRendererWithFS[config.BaseConfig](outputDir, "template_files/system", systemTemplatesToRender)
}

func NewRendererWithFS[T config.BaseConfig | config.DevEnvConfig](outputDir string, templateRoot string, targetTemplates []string) *Renderer[T] {
	return &Renderer[T]{
		outputDir:       outputDir,
		templateRoot:    templateRoot,
		targetTemplates: targetTemplates,
	}
}

func templateFuncs(templateRoot string) template.FuncMap {
	return template.FuncMap{
		"b64enc": func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		},
		"indent": func(spaces int, s string) string {
			padding := strings.Repeat(" ", spaces)
			return strings.ReplaceAll(s, "\n", "\n"+padding)
		},
		"getTemplatedScript": func(scriptName string, config *config.DevEnvConfig) (string, error) {
			// Read the template content
			content, err := templates.ReadFile(filepath.Join(templateRoot, fmt.Sprintf("scripts/templated/%s", scriptName)))
			if err != nil {
				return "", fmt.Errorf("failed to read templated script %s: %w", scriptName, err)
			}

			// Parse and execute template with config
			tmpl, err := template.New(scriptName).Funcs(templateFuncs(templateRoot)).Parse(string(content))
			if err != nil {
				return "", fmt.Errorf("failed to parse script template %s: %w", scriptName, err)
			}

			var output strings.Builder
			if err := tmpl.Execute(&output, config); err != nil {
				return "", fmt.Errorf("failed to render script template %s: %w", scriptName, err)
			}

			return output.String(), nil
		},
		"getStaticScript": func(scriptName string) (string, error) {
			content, err := templates.ReadFile(filepath.Join(templateRoot, fmt.Sprintf("scripts/static/%s", scriptName)))
			if err != nil {
				return "", fmt.Errorf("failed to read static script %s: %w", scriptName, err)
			}
			return string(content), nil
		},
	}
}

func (r *Renderer[T]) RenderTemplate(templateName string, config *T) error {
	// Get the template content from embedded files
	templateContent, err := templates.ReadFile(filepath.Join(r.templateRoot, fmt.Sprintf("manifests/%s.tmpl", templateName)))
	if err != nil {
		return err
	}

	// Parse template
	tmpl, err := template.New(templateName).Funcs(templateFuncs(r.templateRoot)).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	// Output filename is simply template name + .yaml
	outputFilename := fmt.Sprintf("%s.yaml", templateName)

	// Create output file
	outputPath := filepath.Join(r.outputDir, outputFilename)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", filepath.Dir(outputPath), err)
	}
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	// Execute template with DevEnvConfig - simple and clean!
	if err := tmpl.Execute(outputFile, config); err != nil {
		return fmt.Errorf("failed to render template %s: %w", templateName, err)
	}

	fmt.Printf("✅ Generated %s\n", outputPath)
	return nil
}

func (r *Renderer[T]) RenderAll(config *T) error {
	for _, templateName := range r.targetTemplates {
		if err := r.RenderTemplate(templateName, config); err != nil {
			return fmt.Errorf("failed to render template %s: %w", templateName, err)
		}
	}
	return nil
}
