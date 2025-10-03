package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version subcommand
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("devenv version %s\n", version)

		if verbose {
			fmt.Printf("  Build time: %s\n", buildTime)
			fmt.Printf("  Git commit: %s\n", gitCommit)
			fmt.Printf("  Go version: %s\n", goVersion)
		}
	},
}
