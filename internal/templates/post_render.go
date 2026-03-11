package templates

import (
	"fmt"
	"os"
	"path/filepath"
)

type PostRenderOptions struct {
	CleanupUnplanned bool
}

func NewPostRenderOptions(cleanupUnplanned bool) PostRenderOptions {
	return PostRenderOptions{CleanupUnplanned: cleanupUnplanned}
}

// RunPostRender executes post-render steps for a completed render pass.
func RunPostRender(outputDir string, plan RenderPlan, opts PostRenderOptions) error {
	if opts.CleanupUnplanned {
		if err := runPostRenderCleanup(outputDir, plan); err != nil {
			return fmt.Errorf("cleanup unplanned outputs: %w", err)
		}
	}

	return nil
}

// runPostRenderCleanup removes managed outputs that are no longer planned.
func runPostRenderCleanup(outputDir string, plan RenderPlan) error {
	planned := make(map[string]struct{}, len(plan.TemplateNames))
	for _, templateName := range plan.TemplateNames {
		planned[templateName] = struct{}{}
	}

	for _, templateName := range plan.ManagedTemplates {
		if _, ok := planned[templateName]; ok {
			continue
		}
		if err := removeOutput(outputDir, templateName); err != nil {
			return fmt.Errorf("failed to remove output for template %s: %w", templateName, err)
		}
	}

	return nil
}

func removeOutput(outputDir string, templateName string) error {
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", templateName))
	err := os.Remove(outputPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
