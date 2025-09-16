// DevEnv CLI generates Kubernetes manifests for developer environments.
// This command-line tool processes YAML configurations and creates complete
// Kubernetes resources including StatefulSets, Services, and ConfigMaps.
//
// Basic usage:
//
//	devenv generate eywalker
//	devenv generate --all-developers
//	devenv validate eywalker
//
// Use --help with any command for detailed usage information.
package main

import (
	"fmt"
	"os"
)

// Build-time variables (will be set by build system later)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
