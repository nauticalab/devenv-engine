package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nauticalab/devenv-engine/internal/config"
	"github.com/nauticalab/devenv-engine/internal/templates"
)

// DeveloperJob represents work to be done for one developer
type DeveloperJob struct {
	Name string
}

// ProcessingResult represents the outcome of processing one developer
type ProcessingResult struct {
	Developer string
	Success   bool
	Error     error
	Duration  time.Duration
}

// Options holds configuration for the generate command
type GenerateOptions struct {
	OutputDir string
	ConfigDir string
	DryRun    bool
	Verbose   bool
}

// RunAll generates manifests for all developers
func GenerateRunAll(opts GenerateOptions) {
	// Step 1: Load global config once
	globalConfig, err := config.LoadGlobalConfig(opts.ConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading global config in %s: %v\n", opts.ConfigDir, err)
		os.Exit(1)
	}

	if opts.Verbose {
		fmt.Printf("Generating system manifests in %s\n", opts.OutputDir)
	}

	// Step 2: Generate system manifests once
	if !opts.DryRun {
		if err := generateSystemManifests(globalConfig, opts.OutputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating system manifests: %v\n", err)
			os.Exit(1)
		}
	}

	// Step 3: Discover all developers
	developers, err := findAllDevelopers(opts.ConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering developers: %v\n", err)
		os.Exit(1)
	}

	if len(developers) == 0 {
		fmt.Printf("No developers found in %s\n", opts.ConfigDir)
		return
	}

	fmt.Printf("Found %d developers to process.\n", len(developers))

	// Step 4: Set up channels for worker communication
	const numWorkers = 4
	jobs := make(chan DeveloperJob, len(developers))
	results := make(chan ProcessingResult, len(developers))

	// Step 5: Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		go developerWorker(jobs, results, globalConfig, opts)
	}

	// Step 6: Send all jobs to workers
	for _, dev := range developers {
		jobs <- DeveloperJob{Name: dev}
	}
	close(jobs)

	// Step 7: Collect results
	var successCount, failureCount int
	var failures []ProcessingResult

	for i := 0; i < len(developers); i++ {
		result := <-results
		if result.Success {
			successCount++
			fmt.Printf("[%d/%d] ✅ %s (%.1fs)\n",
				i+1, len(developers), result.Developer, result.Duration.Seconds())
		} else {
			failureCount++
			failures = append(failures, result)
			fmt.Printf("[%d/%d] ❌ %s (%.1fs): %v\n",
				i+1, len(developers), result.Developer, result.Duration.Seconds(), result.Error)
		}
	}

	// Step 8: Print final summary
	fmt.Printf("\n🎉 Batch processing complete!\n")
	fmt.Printf("✅ Successful: %d\n", successCount)
	if failureCount > 0 {
		fmt.Printf("❌ Failed: %d\n", failureCount)
	}

	if failureCount > 0 {
		fmt.Printf("\nFailures:\n")
		for _, failure := range failures {
			fmt.Printf("  - %s: %v\n", failure.Developer, failure.Error)
		}
		os.Exit(1) // Exit with error if any failures
	}
}

// RunSingle generates manifests for a single developer
func GenerateRunSingle(developerName string, opts GenerateOptions) {
	fmt.Printf("Generating manifests for developer: %s\n", developerName)

	if opts.Verbose {
		fmt.Printf("Output directory: %s\n", opts.OutputDir)
		fmt.Printf("Config directory: %s\n", opts.ConfigDir)
		fmt.Printf("Dry run mode: %t\n", opts.DryRun)
	}

	userOutputDir := filepath.Join(opts.OutputDir, developerName)

	globalConfig, err := config.LoadGlobalConfig(opts.ConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading global config in %s: %v\n", opts.ConfigDir, err)
		os.Exit(1)
	}

	if err := generateSystemManifests(globalConfig, opts.OutputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating system manifests: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.LoadDeveloperConfigWithBaseConfig(opts.ConfigDir, developerName, globalConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config for developer %s: %v\n", developerName, err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully loaded configuration for developer: %s\n", cfg.Name)

	if opts.Verbose {
		printConfigSummary(cfg)
	}

	if !opts.DryRun {
		if err := generateDeveloperManifests(cfg, userOutputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating manifests: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("🔍 Dry run - would generate manifests to: %s\n", userOutputDir)
	}
}

func developerWorker(jobs <-chan DeveloperJob, results chan<- ProcessingResult, globalConfig *config.BaseConfig, opts GenerateOptions) {
	for job := range jobs {
		startTime := time.Now()
		err := processSingleDeveloperForBatchWithError(job.Name, globalConfig, opts)

		results <- ProcessingResult{
			Developer: job.Name,
			Success:   err == nil,
			Error:     err,
			Duration:  time.Since(startTime),
		}
	}
}

// processSingleDeveloperForBatchWithError processes a single developer for batch mode
func processSingleDeveloperForBatchWithError(developerName string, globalConfig *config.BaseConfig, opts GenerateOptions) error {
	if opts.Verbose {
		fmt.Printf("Processing developer: %s\n", developerName)
	}

	cfg, err := config.LoadDeveloperConfigWithBaseConfig(opts.ConfigDir, developerName, globalConfig)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create user-specific output directory
	userOutputDir := filepath.Join(opts.OutputDir, developerName)

	if !opts.DryRun {
		if err := generateDeveloperManifests(cfg, userOutputDir); err != nil {
			return fmt.Errorf("failed to generate manifests: %w", err)
		}
	}

	return nil
}

func findAllDevelopers(configDir string) ([]string, error) {
	var developers []string

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check to make sure devenv-config.yaml exists in this directory
			configPath := filepath.Join(configDir, entry.Name(), "devenv-config.yaml")
			if _, err := os.Stat(configPath); err == nil {
				developers = append(developers, entry.Name())
			}
		}
	}

	return developers, nil
}

func generateSystemManifests(cfg *config.BaseConfig, outputDir string) error {
	// Create template renderer
	renderer := templates.NewSystemRenderer(outputDir)

	// Render all main templates
	if err := renderer.RenderAll(cfg); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	fmt.Printf("🎉 Successfully generated system manifests\n")

	return nil
}

// generateDeveloperManifests creates Kubernetes manifests for a developer
func generateDeveloperManifests(cfg *config.DevEnvConfig, outputDir string) error {
	// Create template renderer
	renderer := templates.NewDevRenderer(outputDir)

	// Render all main templates
	if err := renderer.RenderAll(cfg); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	fmt.Printf("🎉 Successfully generated manifests for %s\n", cfg.Name)

	return nil
}

// Helper function to print config summary
func printConfigSummary(cfg *config.DevEnvConfig) {
	fmt.Printf("\nConfiguration Summary:\n")
	fmt.Printf("  Name: %s\n", cfg.Name)

	sshKeys, _ := cfg.GetSSHKeys()
	fmt.Printf("  SSH Keys: %d configured\n", len(sshKeys))

	if cfg.SSHPort != 0 {
		fmt.Printf("  SSH Port: %d\n", cfg.SSHPort)
	}

	if cfg.Git.Name != "" {
		fmt.Printf("  Git: %s <%s>\n", cfg.Git.Name, cfg.Git.Email)
	}

	cpuStr := cfg.CPU()    // e.g., "4000m" or "0"
	memStr := cfg.Memory() // e.g., "16Gi" or ""

	hasCPU := cpuStr != "0"
	hasMem := memStr != ""

	if hasCPU || hasMem {
		var parts []string
		if hasCPU {
			parts = append(parts, fmt.Sprintf("CPU=%s", cpuStr))
		}
		if hasMem {
			parts = append(parts, fmt.Sprintf("Memory=%s", memStr))
		}
		fmt.Printf("  Resources: %s\n", strings.Join(parts, ", "))
	}

	if len(cfg.Volumes) > 0 {
		fmt.Printf("  Volumes: %d configured\n", len(cfg.Volumes))
	}

	fmt.Printf("  Developer Config Dir: %s\n", cfg.GetDeveloperDir())
}
