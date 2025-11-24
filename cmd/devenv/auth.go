package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nauticalab/devenv-engine/internal/manager/auth"
	"github.com/nauticalab/devenv-engine/internal/manager/client"
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
		config, err := LoadCLIConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if config.ManagerURL == "" {
			fmt.Fprintf(os.Stderr, "Error: manager URL is required. Set DEVEN_MANAGER_URL env var or configure in ~/.devenv/config.yaml\n")
			os.Exit(1)
		}

		// Create manager client
		authProvider := auth.NewK8sSAProvider(nil, "", "", "")
		c := client.NewClient(config.ManagerURL, authProvider)

		// Get identity
		whoami, err := c.WhoAmI(context.Background())
		if err != nil {
			fmt.Printf("Error getting authentication info: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Authenticated as: %s\n", whoami.Username)
		fmt.Printf("Type: %s\n", whoami.Type)
		fmt.Printf("Developer: %s\n", whoami.Developer)
		if whoami.Namespace != "" {
			fmt.Printf("Namespace: %s\n", whoami.Namespace)
		}
	},
}

func init() {
	// Add subcommands
	authCmd.AddCommand(authListCmd)
}
