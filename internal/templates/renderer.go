package templates

import (
	"embed"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/walkerlab/devenv-engine/internal/config"
)

// Embed all templates at compile time
//
//go:embed files/*.tmpl
var templates embed.FS

var templatesToRender = []string{"statefulset", "service", "env-vars", "startup-scripts", "secret"}

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

	// Execute template with DevEnvConfig
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
