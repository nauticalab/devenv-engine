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

var templatesToRender = []string{"statefulset", "service", "env-vars",
	"startup-scripts", "ingress"}

// Embed all templates and scripts at compile time
//
//go:embed files/*.tmpl
var templates embed.FS

//go:embed scripts/templated/*
var templatedScripts embed.FS

//go:embed scripts/static/*
var staticScripts embed.FS

// Renderer handles template operations
type Renderer struct {
	outputDir string
}

// NewRenderer creates a new template renderer
func NewRenderer(outputDir string) *Renderer {
	return &Renderer{
		outputDir: outputDir,
	}
}

func templateFuncs() template.FuncMap {
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
			content, err := templatedScripts.ReadFile(fmt.Sprintf("scripts/templated/%s", scriptName))
			if err != nil {
				return "", fmt.Errorf("failed to read templated script %s: %w", scriptName, err)
			}

			// Parse and execute template with config
			tmpl, err := template.New(scriptName).Funcs(templateFuncs()).Parse(string(content))
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
			content, err := staticScripts.ReadFile(fmt.Sprintf("scripts/static/%s", scriptName))
			if err != nil {
				return "", fmt.Errorf("failed to read static script %s: %w", scriptName, err)
			}
			return string(content), nil
		},
	}
}

func (r *Renderer) RenderTemplate(templateName string, config *config.DevEnvConfig) error {
	// Get the template content from embedded files
	templateContent, err := templates.ReadFile(fmt.Sprintf("files/%s.tmpl", templateName))
	if err != nil {
		return err
	}

	// Parse template
	tmpl, err := template.New(templateName).Funcs(templateFuncs()).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", r.outputDir, err)
	}

	// Output filename is simply template name + .yaml
	outputFilename := fmt.Sprintf("%s.yaml", templateName)

	// Create output file
	outputPath := filepath.Join(r.outputDir, outputFilename)
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

func (r *Renderer) RenderAll(config *config.DevEnvConfig) error {
	for _, templateName := range templatesToRender {
		if err := r.RenderTemplate(templateName, config); err != nil {
			return fmt.Errorf("failed to render template %s: %w", templateName, err)
		}
	}
	return nil
}
