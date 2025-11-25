package client

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nauticalab/devenv-engine/internal/cli"
	"github.com/nauticalab/devenv-engine/internal/manager/api"
	"github.com/nauticalab/devenv-engine/internal/manager/auth"
	"github.com/nauticalab/devenv-engine/internal/manager/client"
)

// Options holds configuration for the pods command
type PodsOptions struct {
	Namespace     string
	AllNamespaces bool
	LabelFilter   string
	ShowAll       bool
	ManagerURL    string
	SATokenPath   string
}

// RunList lists running pods
func RunListPods(args []string, opts PodsOptions) error {
	ctx := context.Background()

	// Create manager client
	authProvider := auth.NewK8sSAProvider(nil, "", "", opts.SATokenPath)
	c := client.NewClient(opts.ManagerURL, authProvider)

	// List pods
	resp, err := c.ListPods(ctx, opts.Namespace, opts.AllNamespaces)
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	// Filter pods locally if needed
	var filteredPods []api.Pod
	for _, pod := range resp.Pods {
		if opts.ShowAll || pod.Status == "Running" {
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

// RunDelete deletes a pod
func RunDeletePod(podName string, opts PodsOptions) error {
	ctx := context.Background()

	// Create manager client
	authProvider := auth.NewK8sSAProvider(nil, "", "", opts.SATokenPath)
	c := client.NewClient(opts.ManagerURL, authProvider)

	// Delete pod
	resp, err := c.DeletePod(ctx, opts.Namespace, podName)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	if resp.Success {
		fmt.Printf("✓ Pod '%s' deleted from namespace '%s'\n", podName, opts.Namespace)
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

// LoadConfig loads the CLI configuration and applies overrides
func LoadConfig(managerURLOverride string) (*cli.CLIConfig, error) {
	// Load configuration
	cfg, err := cli.LoadCLIConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with flag if provided
	if managerURLOverride != "" {
		cfg.ManagerURL = managerURLOverride
	}

	if cfg.ManagerURL == "" {
		return nil, fmt.Errorf("manager URL is required. Set DEVEN_MANAGER_URL env var, use --manager-url flag, or configure in ~/.devenv/config.yaml")
	}

	return cfg, nil
}
