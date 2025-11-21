package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nauticalab/devenv-engine/internal/api"
	"github.com/nauticalab/devenv-engine/internal/k8s"
	"github.com/spf13/cobra"
)

// ServerConfig holds the configuration for the server
type ServerConfig struct {
	Port     int
	Bind     string
	Audience string
}

var serverConfig ServerConfig

var serverCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the DevENV Manager HTTP API server",
	Long: `Start the DevENV Manager HTTP API server.

The server provides an HTTP API for managing developer environments:
  - List pods for authenticated developer
  - Delete pods owned by authenticated developer
  - Health and version endpoints

Authentication is performed using Kubernetes service account tokens.
Clients must provide a Bearer token in the Authorization header.`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().IntVarP(&serverConfig.Port, "port", "p", 8080, "Port to listen on")
	serverCmd.Flags().StringVarP(&serverConfig.Bind, "bind", "b", "0.0.0.0", "Address to bind to")
	serverCmd.Flags().StringVar(&serverConfig.Audience, "audience", "devenv-manager", "Expected token audience for service account tokens")
}

func runServer(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create K8s client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Build server configuration
	config := api.ServerConfig{
		Port:      serverConfig.Port,
		Audience:  serverConfig.Audience,
		K8sClient: k8sClient,
		Version:   version,
		GitCommit: gitCommit,
		BuildTime: buildTime,
		GoVersion: "", // Not needed for now, can be added to build vars later
	}

	// Create the API server
	server, err := api.NewServer(config)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", serverConfig.Bind, serverConfig.Port)
	fmt.Printf("Starting DevENV Manager API server on %s\n", addr)
	fmt.Printf("Token audience: %s\n", serverConfig.Audience)
	fmt.Printf("\nEndpoints:\n")
	fmt.Printf("  GET  /api/v1/health          - Health check\n")
	fmt.Printf("  GET  /api/v1/version         - Version information\n")
	fmt.Printf("  GET  /api/v1/pods            - List pods (authenticated)\n")
	fmt.Printf("  DELETE /api/v1/pods/{ns}/{name} - Delete pod (authenticated)\n")
	fmt.Printf("\n")

	// Start the server with graceful shutdown
	if err := server.StartWithContext(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	fmt.Println("Server shutdown complete")
	return nil
}
