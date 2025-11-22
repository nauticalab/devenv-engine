package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nauticalab/devenv-engine/internal/manager"
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
		// Create manager client
		client := manager.NewClient(manager.ClientConfig{
			BaseURL: os.Getenv("DEVEN_MANAGER_URL"),
		})

		// Get identity
		whoami, err := client.WhoAmI(context.Background())
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
	// Add auth command to root
	rootCmd.AddCommand(authCmd)

	// Add subcommands
	authCmd.AddCommand(authListCmd)
}
