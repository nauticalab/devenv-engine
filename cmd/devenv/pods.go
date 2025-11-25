package main

import (
	"fmt"
	"os"

	clientcmd "github.com/nauticalab/devenv-engine/internal/cli/client"
	"github.com/spf13/cobra"
)

var (
	// Pods command flags
	podsNamespace    string
	podsAllNamespace bool
	podsLabelFilter  string
	podsShowAll      bool
	podsManagerURL   string
)

var podsCmd = &cobra.Command{
	Use:   "pods",
	Short: "Manage and inspect developer environment pods",
	Long: `Interact with Kubernetes pods for developer environments via the Devenv Manager.

Use subcommands to list, inspect, or manage pods.`,
}

var podsListCmd = &cobra.Command{
	Use:   "list [developer-name]",
	Short: "List running pods",
	Long: `List running pods in the Kubernetes cluster via the Devenv Manager.

Examples:
  # List all pods in the default namespace
  devenv pods list

  # List pods for a specific developer
  devenv pods list eywalker

  # List all pods across all namespaces
  devenv pods list --all-namespaces

  # List pods in a specific namespace
  devenv pods list --namespace devenv

  # List only running pods
  devenv pods list --show-all=false`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := clientcmd.LoadConfig(podsManagerURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		opts := clientcmd.PodsOptions{
			Namespace:     podsNamespace,
			AllNamespaces: podsAllNamespace,
			LabelFilter:   podsLabelFilter,
			ShowAll:       podsShowAll,
			ManagerURL:    cfg.ManagerURL,
			SATokenPath:   cfg.SATokenPath,
		}

		if err := clientcmd.RunListPods(args, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var podsDeleteCmd = &cobra.Command{
	Use:   "delete [pod-name]",
	Short: "Delete a pod",
	Long: `Delete a pod by name via the Devenv Manager.

Examples:
  # Delete a pod in the default namespace
  devenv pods delete my-pod

  # Delete a pod in a specific namespace
  devenv pods delete my-pod --namespace devenv`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := clientcmd.LoadConfig(podsManagerURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		opts := clientcmd.PodsOptions{
			Namespace:   podsNamespace,
			ManagerURL:  cfg.ManagerURL,
			SATokenPath: cfg.SATokenPath,
		}

		podName := args[0]
		if err := clientcmd.RunDeletePod(podName, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Add pods command to root
	rootCmd.AddCommand(podsCmd)

	// Add pods command flags
	podsListCmd.Flags().StringVarP(&podsNamespace, "namespace", "n", "", "Kubernetes namespace (default: default)")
	podsListCmd.Flags().BoolVarP(&podsAllNamespace, "all-namespaces", "A", false, "List pods across all namespaces")
	podsListCmd.Flags().StringVarP(&podsLabelFilter, "labels", "l", "", "Filter pods by label selector (e.g., app=devenv)")
	podsListCmd.Flags().BoolVar(&podsShowAll, "show-all", true, "Show all pods (not just running)")
	podsListCmd.Flags().StringVar(&podsManagerURL, "manager-url", "", "URL of the Devenv Manager API")

	podsDeleteCmd.Flags().StringVarP(&podsNamespace, "namespace", "n", "default", "Kubernetes namespace")
	podsDeleteCmd.Flags().StringVar(&podsManagerURL, "manager-url", "", "URL of the Devenv Manager API")

	// Add subcommands
	podsCmd.AddCommand(podsListCmd)
	podsCmd.AddCommand(podsDeleteCmd)
}
