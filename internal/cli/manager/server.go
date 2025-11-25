package manager

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nauticalab/devenv-engine/internal/k8s"
	"github.com/nauticalab/devenv-engine/internal/manager/api"
)

// ServerOptions holds configuration for the manager server command
type ServerOptions struct {
	Port        int
	Bind        string
	Audience    string
	TLSCertPath string
	TLSKeyPath  string
	Version     string
	BuildTime   string
	GitCommit   string
	GoVersion   string
}

// RunServer starts the manager server
func RunServer(opts ServerOptions) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create K8s client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Build server configuration
	config := api.ServerConfig{
		Port:        opts.Port,
		Audience:    opts.Audience,
		K8sClient:   k8sClient,
		Version:     opts.Version,
		GitCommit:   opts.GitCommit,
		BuildTime:   opts.BuildTime,
		GoVersion:   opts.GoVersion,
		TLSCertPath: opts.TLSCertPath,
		TLSKeyPath:  opts.TLSKeyPath,
	}

	// Create the API server
	server, err := api.NewServer(config)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", opts.Bind, opts.Port)
	fmt.Printf("Starting DevENV Manager API server on %s\n", addr)
	fmt.Printf("Token audience: %s\n", opts.Audience)
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
