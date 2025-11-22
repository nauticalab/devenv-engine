package main

import (
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: " developer environment pods",
	Long:  `List and manage developer environment pods.`,
}
var authInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get authentication information",
	Long:  `Get authentication information.`,
	RunE:  runAuthInfo,
}

func init() {
	// Add auth command to root
	rootCmd.AddCommand(authCmd)

	// Add subcommands
	authCmd.AddCommand(authInfoCmd)
}

func runAuthInfo(cmd *cobra.Command, args []string) error {
	return nil
}
