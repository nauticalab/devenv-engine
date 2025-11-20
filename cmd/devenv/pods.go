package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/nauticalab/devenv-engine/internal/api"
	"github.com/nauticalab/devenv-engine/internal/k8s"
	"github.com/nauticalab/devenv-engine/internal/manager"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

var (
	// Pods command flags
	podsRemote       bool
	podsManagerURL   string
	podsNamespace    string
	podsTokenPath    string
	podsAllNamespace bool
)

var podsCmd = &cobra.Command{
	Use:   "pods",
	Short: "List and manage developer environment pods",
	Long: `List and manage developer environment pods.

By default, connects directly to Kubernetes API.
Use --remote to connect via the DevEnv Manager API server.`,
}

var podsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your developer environment pods",
	Long: `List your developer environment pods.

Examples:
  # List pods via direct Kubernetes access
  devenv pods list

  # List pods via manager API
  devenv pods list --remote --manager-url http://manager-service:8080

  # List pods in specific namespace
  devenv pods list --namespace devenv`,
	RunE: runPodsList,
}

var podsDeleteCmd = &cobra.Command{
	Use:   "delete <pod-name>",
	Short: "Delete a developer environment pod",
	Long: `Delete a developer environment pod.

Examples:
  # Delete a pod via direct Kubernetes access
  devenv pods delete my-pod --namespace devenv

  # Delete a pod via manager API
  devenv pods delete my-pod --remote --manager-url http://manager-service:8080 --namespace devenv`,
	Args: cobra.ExactArgs(1),
	RunE: runPodsDelete,
}

func init() {
	// Add pods command to root
	rootCmd.AddCommand(podsCmd)

	// Add subcommands
	podsCmd.AddCommand(podsListCmd)
	podsCmd.AddCommand(podsDeleteCmd)

	// Flags for list command
	podsListCmd.Flags().BoolVar(&podsRemote, "remote", false, "Use manager API instead of direct Kubernetes access")
	podsListCmd.Flags().StringVar(&podsManagerURL, "manager-url", "", "Manager API URL (required when --remote is set)")
	podsListCmd.Flags().StringVarP(&podsNamespace, "namespace", "n", "", "Kubernetes namespace (empty for all namespaces)")
	podsListCmd.Flags().StringVar(&podsTokenPath, "token-path", manager.DefaultTokenPath, "Path to service account token")
	podsListCmd.Flags().BoolVarP(&podsAllNamespace, "all-namespaces", "A", false, "List pods across all namespaces (direct mode only)")

	// Flags for delete command
	podsDeleteCmd.Flags().BoolVar(&podsRemote, "remote", false, "Use manager API instead of direct Kubernetes access")
	podsDeleteCmd.Flags().StringVar(&podsManagerURL, "manager-url", "", "Manager API URL (required when --remote is set)")
	podsDeleteCmd.Flags().StringVarP(&podsNamespace, "namespace", "n", "default", "Kubernetes namespace")
	podsDeleteCmd.Flags().StringVar(&podsTokenPath, "token-path", manager.DefaultTokenPath, "Path to service account token")
}

func runPodsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if podsRemote {
		return listPodsRemote(ctx)
	}
	return listPodsDirect(ctx)
}

func runPodsDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	podName := args[0]

	if podsRemote {
		return deletePodRemote(ctx, podName)
	}
	return deletePodDirect(ctx, podName)
}

// listPodsRemote lists pods via the manager API
func listPodsRemote(ctx context.Context) error {
	if podsManagerURL == "" {
		return fmt.Errorf("--manager-url is required when using --remote")
	}

	if verbose {
		fmt.Printf("Connecting to manager API at %s\n", podsManagerURL)
	}

	// Create manager client
	client := manager.NewClient(manager.ClientConfig{
		BaseURL:   podsManagerURL,
		TokenPath: podsTokenPath,
	})

	// List pods
	resp, err := client.ListPods(ctx, podsNamespace)
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(resp.Pods) == 0 {
		fmt.Println("No pods found")
		return nil
	}

	// Print pods table
	printPodsTableRemote(resp.Pods)
	return nil
}

// listPodsDirect lists pods via direct Kubernetes access
func listPodsDirect(ctx context.Context) error {
	if verbose {
		fmt.Println("Connecting to Kubernetes API directly...")
	}

	// Create Kubernetes client
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	var pods []PodInfo

	// List pods based on flags
	if podsAllNamespace {
		podList, err := client.ListAllPods(ctx)
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}
		for _, pod := range podList.Items {
			pods = append(pods, PodInfo{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    k8s.GetPodStatus(&pod),
				Restarts:  getTotalRestarts(&pod),
				Age:       formatAge(pod.CreationTimestamp.Time),
				Developer: pod.Labels["developer"],
				NodeName:  pod.Spec.NodeName,
				PodIP:     pod.Status.PodIP,
			})
		}
	} else {
		ns := podsNamespace
		if ns == "" {
			ns = "default"
		}
		podList, err := client.ListPods(ctx, ns)
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}
		for _, pod := range podList.Items {
			pods = append(pods, PodInfo{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    k8s.GetPodStatus(&pod),
				Restarts:  getTotalRestarts(&pod),
				Age:       formatAge(pod.CreationTimestamp.Time),
				Developer: pod.Labels["developer"],
				NodeName:  pod.Spec.NodeName,
				PodIP:     pod.Status.PodIP,
			})
		}
	}

	if len(pods) == 0 {
		fmt.Println("No pods found")
		return nil
	}

	// Print pods table
	printPodsTableDirect(pods, podsAllNamespace)
	return nil
}

// deletePodRemote deletes a pod via the manager API
func deletePodRemote(ctx context.Context, podName string) error {
	if podsManagerURL == "" {
		return fmt.Errorf("--manager-url is required when using --remote")
	}

	if verbose {
		fmt.Printf("Connecting to manager API at %s\n", podsManagerURL)
	}

	// Create manager client
	client := manager.NewClient(manager.ClientConfig{
		BaseURL:   podsManagerURL,
		TokenPath: podsTokenPath,
	})

	// Delete pod
	resp, err := client.DeletePod(ctx, podsNamespace, podName)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	if resp.Success {
		fmt.Printf("✓ %s\n", resp.Message)
	} else {
		fmt.Printf("✗ %s\n", resp.Message)
	}

	return nil
}

// deletePodDirect deletes a pod via direct Kubernetes access
func deletePodDirect(ctx context.Context, podName string) error {
	if verbose {
		fmt.Println("Connecting to Kubernetes API directly...")
	}

	// Create Kubernetes client
	client, err := k8s.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Delete pod
	if err := client.DeletePod(ctx, podsNamespace, podName); err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	fmt.Printf("✓ Pod '%s' deleted from namespace '%s'\n", podName, podsNamespace)
	return nil
}

// printPodsTableRemote prints pods from manager API response
func printPodsTableRemote(pods []api.Pod) {
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

// PodInfo represents pod information for direct access
type PodInfo struct {
	Name      string
	Namespace string
	Status    string
	Restarts  int32
	Age       string
	Developer string
	NodeName  string
	PodIP     string
}

// printPodsTableDirect prints pods from direct Kubernetes access
func printPodsTableDirect(pods []PodInfo, allNamespaces bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Print header
	if allNamespaces {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tSTATUS\tRESTARTS\tAGE\tDEVELOPER")
	} else {
		fmt.Fprintln(w, "NAME\tSTATUS\tRESTARTS\tAGE\tDEVELOPER")
	}

	// Print each pod
	for _, pod := range pods {
		developer := pod.Developer
		if developer == "" {
			developer = "-"
		}

		if allNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				pod.Namespace, pod.Name, pod.Status, pod.Restarts, pod.Age, developer)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				pod.Name, pod.Status, pod.Restarts, pod.Age, developer)
		}
	}
}

// getTotalRestarts calculates total container restarts
func getTotalRestarts(pod *corev1.Pod) int32 {
	var total int32
	for _, cs := range pod.Status.ContainerStatuses {
		total += cs.RestartCount
	}
	return total
}

// formatAge formats a duration as a human-readable age string
func formatAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	days := int(duration.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}
