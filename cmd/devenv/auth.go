package main

import (
	"fmt"
	"os"

	"github.com/nauticalab/devenv-engine/internal/cli"
	"github.com/nauticalab/devenv-engine/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  `Manage authentication for the DevENV engine.`,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List current authentication information",
	Long:  `Display the current authentication information, including the authenticated user and their developer identity.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.LoadCLIConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		cli.AuthRunList(cfg.ManagerURL)
	},
}

func init() {
	// Add subcommands
	authCmd.AddCommand(authListCmd)
}
