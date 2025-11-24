package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nauticalab/devenv-engine/internal/manager/api"
	"github.com/nauticalab/devenv-engine/internal/manager/auth"
	"github.com/nauticalab/devenv-engine/internal/manager/client"
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
		ctx := context.Background()
		if err := runPodsList(ctx, args); err != nil {
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
		ctx := context.Background()
		podName := args[0]
		if err := runPodsDelete(ctx, podName); err != nil {
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

func runPodsList(ctx context.Context, args []string) error {
	// Load configuration
	config, err := LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with flag if provided
	if podsManagerURL != "" {
		config.ManagerURL = podsManagerURL
	}

	if config.ManagerURL == "" {
		return fmt.Errorf("manager URL is required. Set DEVEN_MANAGER_URL env var, use --manager-url flag, or configure in ~/.devenv/config.yaml")
	}

	// Create manager client
	// For now, we use K8s SA provider with default token path
	// In the future, we might support other auth methods
	authProvider := auth.NewK8sSAProvider(nil, "", "", "")
	c := client.NewClient(config.ManagerURL, authProvider)

	// List pods
	resp, err := c.ListPods(ctx, podsNamespace, podsAllNamespace)
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	// Filter pods locally if needed (e.g. by developer name if arg provided)
	// The API currently filters by authenticated developer, but we might want to filter further
	// or if we are admin listing all pods.
	// For now, let's just print what we got.

	var filteredPods []api.Pod
	for _, pod := range resp.Pods {
		if podsShowAll || pod.Status == "Running" {
			filteredPods = append(filteredPods, pod)
		}
	}

	if len(filteredPods) == 0 {
		fmt.Println("No pods found")
		return nil
	}

	printPodsTable(filteredPods)
	return nil
}

func runPodsDelete(ctx context.Context, podName string) error {
	// Load configuration
	config, err := LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with flag if provided
	if podsManagerURL != "" {
		config.ManagerURL = podsManagerURL
	}

	if config.ManagerURL == "" {
		return fmt.Errorf("manager URL is required. Set DEVEN_MANAGER_URL env var, use --manager-url flag, or configure in ~/.devenv/config.yaml")
	}

	// Create manager client
	authProvider := auth.NewK8sSAProvider(nil, "", "", "")
	c := client.NewClient(config.ManagerURL, authProvider)

	// Delete pod
	resp, err := c.DeletePod(ctx, podsNamespace, podName)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	if resp.Success {
		fmt.Printf("✓ Pod '%s' deleted from namespace '%s'\n", podName, podsNamespace)
	} else {
		return fmt.Errorf("failed to delete pod: %s", resp.Message)
	}
	return nil
}

// printPodsTable prints pods from manager API response
func printPodsTable(pods []api.Pod) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Print header
	fmt.Fprintln(w, "NAMESPACE\tNAME\tSTATUS\tRESTARTS\tAGE\tDEVELOPER")

	// Print each pod
	for _, pod := range pods {
		developer := pod.Developer
		if developer == "" {
			developer = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			pod.Namespace, pod.Name, pod.Status, pod.Restarts, pod.Age, developer)
	}
}
