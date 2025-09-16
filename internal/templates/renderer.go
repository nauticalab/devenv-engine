package templates

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/walkerlab/devenv-engine/internal/config"
)

// Embed all templates at compile time
//
//go:embed *.tmpl
var templates embed.FS

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

func (r *Renderer) RenderTemplate(templateName string, cfg *config.DevEnvConfig) error {

	// Read from embedded filesystem
	templateContent, err := templates.ReadFile(templateName + ".tmpl")
	if err != nil {
		return err
	}

	// Parse template
	tmpl, err := template.New(templateName).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", r.outputDir, err)
	}

	// Create output file
	outputPath := filepath.Join(r.outputDir, templateName+".yaml")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	// Execute template with DevEnvConfig
	if err := tmpl.Execute(outputFile, cfg); err != nil {
		return fmt.Errorf("failed to render template %s: %w", templateName, err)
	}

	fmt.Printf("✅ Generated %s\n", outputPath)
	return nil
}
