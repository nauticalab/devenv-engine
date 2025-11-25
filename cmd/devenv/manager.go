package main

import (
	"github.com/nauticalab/devenv-engine/internal/cli/manager"
	"github.com/spf13/cobra"
)

// managerCmd represents the manager command
var managerCmd = &cobra.Command{
	Use:   "manager",
	Short: "Manage developer environments in Kubernetes",
	Long: `DevENV Manager provides tools for managing and inspecting developer 
environments running in Kubernetes clusters.

Use this tool to list pods, check statuses, and perform administrative tasks
on developer environments.`,
}

// --- Pods Command ---

var (
	// Pods command flags
	mgrPodsNamespace    string
	mgrPodsAllNamespace bool
	mgrPodsLabelFilter  string
	mgrPodsShowAll      bool
)

var managerPodsCmd = &cobra.Command{
	Use:   "pods",
	Short: "Manage and inspect developer environment pods",
	Long: `Interact with Kubernetes pods for developer environments.

Use subcommands to list, inspect, or manage pods.`,
}

var managerPodsListCmd = &cobra.Command{
	Use:   "list [developer-name]",
	Short: "List running pods",
	Long: `List running pods in the Kubernetes cluster.

Examples:
  # List all pods in the default namespace
  devenv manager pods list

  # List pods for a specific developer
  devenv manager pods list eywalker

  # List all pods across all namespaces
  devenv manager pods list --all-namespaces

  # List pods in a specific namespace
  devenv manager pods list --namespace devenv

  # List only running pods
  devenv manager pods list --show-all=false`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var developerName string
		if len(args) > 0 {
			developerName = args[0]
		}

		opts := manager.PodsOptions{
			Namespace:     mgrPodsNamespace,
			AllNamespaces: mgrPodsAllNamespace,
			LabelFilter:   mgrPodsLabelFilter,
			ShowAll:       mgrPodsShowAll,
			Verbose:       verbose,
		}

		manager.RunListPods(developerName, opts)
	},
}

// --- Server Command ---

// ManagerServerConfig holds the configuration for the server
type ManagerServerConfig struct {
	Port        int
	Bind        string
	Audience    string
	TLSCertPath string
	TLSKeyPath  string
}

var mgrServerConfig ManagerServerConfig

var managerServerCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the DevENV Manager HTTP API server",
	Long: `Start the DevENV Manager HTTP API server.

The server provides an HTTP API for managing developer environments:
  - List pods for authenticated developer
  - Delete pods owned by authenticated developer
  - Health and version endpoints

Authentication is performed using Kubernetes service account tokens.
Clients must provide a Bearer token in the Authorization header.`,
	RunE: runManagerServer,
}

func init() {
	// Add manager command to root
	rootCmd.AddCommand(managerCmd)

	// --- Pods Command Init ---
	managerPodsListCmd.Flags().StringVarP(&mgrPodsNamespace, "namespace", "n", "default", "Kubernetes namespace")
	managerPodsListCmd.Flags().BoolVarP(&mgrPodsAllNamespace, "all-namespaces", "A", false, "List pods across all namespaces")
	managerPodsListCmd.Flags().StringVarP(&mgrPodsLabelFilter, "labels", "l", "", "Filter pods by label selector (e.g., app=devenv)")
	managerPodsListCmd.Flags().BoolVar(&mgrPodsShowAll, "show-all", true, "Show all pods (not just running)")

	managerPodsCmd.AddCommand(managerPodsListCmd)
	managerCmd.AddCommand(managerPodsCmd)

	// --- Server Command Init ---
	managerServerCmd.Flags().IntVarP(&mgrServerConfig.Port, "port", "p", 8080, "Port to listen on")
	managerServerCmd.Flags().StringVarP(&mgrServerConfig.Bind, "bind", "b", "0.0.0.0", "Address to bind to")
	managerServerCmd.Flags().StringVar(&mgrServerConfig.Audience, "audience", "devenv-manager", "Expected token audience for service account tokens")
	managerServerCmd.Flags().StringVar(&mgrServerConfig.TLSCertPath, "tls-cert", "/certs/tls.crt", "Path to TLS certificate")
	managerServerCmd.Flags().StringVar(&mgrServerConfig.TLSKeyPath, "tls-key", "/certs/tls.key", "Path to TLS private key")

	managerCmd.AddCommand(managerServerCmd)
}

func runManagerServer(cmd *cobra.Command, args []string) error {
	opts := manager.ServerOptions{
		Port:        mgrServerConfig.Port,
		Bind:        mgrServerConfig.Bind,
		Audience:    mgrServerConfig.Audience,
		TLSCertPath: mgrServerConfig.TLSCertPath,
		TLSKeyPath:  mgrServerConfig.TLSKeyPath,
		Version:     version,
		BuildTime:   buildTime,
		GitCommit:   gitCommit,
		GoVersion:   goVersion,
	}

	return manager.RunServer(opts)
}
