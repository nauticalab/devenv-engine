package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nauticalab/devenv-engine/internal/k8s"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

var (
	// Pods command flags
	podsNamespace    string
	podsAllNamespace bool
	podsLabelFilter  string
	podsShowAll      bool
)

var podsCmd = &cobra.Command{
	Use:   "pods",
	Short: "Manage and inspect developer environment pods",
	Long: `Interact with Kubernetes pods for developer environments.

Use subcommands to list, inspect, or manage pods.`,
}

var podsListCmd = &cobra.Command{
	Use:   "list [developer-name]",
	Short: "List running pods",
	Long: `List running pods in the Kubernetes cluster.

Examples:
  # List all pods in the default namespace
  manager pods list

  # List pods for a specific developer
  manager pods list eywalker

  # List all pods across all namespaces
  manager pods list --all-namespaces

  # List pods in a specific namespace
  manager pods list --namespace devenv

  # List only running pods
  manager pods list --show-all=false`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// Create Kubernetes client
		client, err := k8s.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
			fmt.Fprintf(os.Stderr, "Make sure you have kubectl configured and can access the cluster.\n")
			os.Exit(1)
		}

		var pods *corev1.PodList
		var developerName string

		if len(args) > 0 {
			developerName = args[0]
		}

		// Determine how to list pods
		if podsAllNamespace {
			if verbose {
				fmt.Println("Listing pods across all namespaces...")
			}
			pods, err = client.ListAllPods(ctx)
		} else if developerName != "" {
			// Filter by developer name using label selector
			labelSelector := fmt.Sprintf("developer=%s", developerName)
			if podsLabelFilter != "" {
				labelSelector = fmt.Sprintf("%s,%s", labelSelector, podsLabelFilter)
			}
			if verbose {
				fmt.Printf("Listing pods for developer: %s (label: %s)\n", developerName, labelSelector)
			}
			pods, err = client.ListPodsWithLabels(ctx, podsNamespace, labelSelector)
		} else if podsLabelFilter != "" {
			if verbose {
				fmt.Printf("Listing pods with labels: %s\n", podsLabelFilter)
			}
			pods, err = client.ListPodsWithLabels(ctx, podsNamespace, podsLabelFilter)
		} else {
			if verbose {
				fmt.Printf("Listing pods in namespace: %s\n", podsNamespace)
			}
			pods, err = client.ListPods(ctx, podsNamespace)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing pods: %v\n", err)
			os.Exit(1)
		}

		// Filter pods if needed
		var filteredPods []corev1.Pod
		for _, pod := range pods.Items {
			if podsShowAll || k8s.IsPodRunning(&pod) {
				filteredPods = append(filteredPods, pod)
			}
		}

		if len(filteredPods) == 0 {
			if developerName != "" {
				fmt.Printf("No pods found for developer: %s\n", developerName)
			} else {
				fmt.Println("No pods found")
			}
			return
		}

		// Print pods in a table format
		printPodsTable(filteredPods)
	},
}

func init() {
	// Add pods command flags
	podsListCmd.Flags().StringVarP(&podsNamespace, "namespace", "n", "default", "Kubernetes namespace")
	podsListCmd.Flags().BoolVarP(&podsAllNamespace, "all-namespaces", "A", false, "List pods across all namespaces")
	podsListCmd.Flags().StringVarP(&podsLabelFilter, "labels", "l", "", "Filter pods by label selector (e.g., app=devenv)")
	podsListCmd.Flags().BoolVar(&podsShowAll, "show-all", true, "Show all pods (not just running)")

	// Add list subcommand to pods command
	podsCmd.AddCommand(podsListCmd)
}

func printPodsTable(pods []corev1.Pod) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Print header
	if podsAllNamespace {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tSTATUS\tRESTARTS\tAGE\tDEVELOPER")
	} else {
		fmt.Fprintln(w, "NAME\tSTATUS\tRESTARTS\tAGE\tDEVELOPER")
	}

	// Print each pod
	for _, pod := range pods {
		status := k8s.GetPodStatus(&pod)
		restarts := getTotalRestarts(&pod)
		age := formatAge(pod.CreationTimestamp.Time)
		developer := pod.Labels["developer"]
		if developer == "" {
			developer = "-"
		}

		if podsAllNamespace {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				pod.Namespace, pod.Name, status, restarts, age, developer)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				pod.Name, status, restarts, age, developer)
		}
	}
}

// getTotalRestarts calculates the total number of container restarts in a pod
func getTotalRestarts(pod *corev1.Pod) int32 {
	var total int32
	for _, cs := range pod.Status.ContainerStatuses {
		total += cs.RestartCount
	}
	return total
}

// formatAge returns a human-readable duration string
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

// printPodDetails prints detailed information about a single pod (for future use)
func printPodDetails(pod *corev1.Pod) {
	fmt.Printf("Name: %s\n", pod.Name)
	fmt.Printf("Namespace: %s\n", pod.Namespace)
	fmt.Printf("Status: %s\n", k8s.GetPodStatus(pod))
	fmt.Printf("IP: %s\n", pod.Status.PodIP)
	fmt.Printf("Node: %s\n", pod.Spec.NodeName)
	fmt.Printf("Created: %s\n", pod.CreationTimestamp.Format(time.RFC3339))

	if len(pod.Labels) > 0 {
		fmt.Println("\nLabels:")
		for key, value := range pod.Labels {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if len(pod.Spec.Containers) > 0 {
		fmt.Println("\nContainers:")
		for _, container := range pod.Spec.Containers {
			fmt.Printf("  - %s (image: %s)\n", container.Name, container.Image)
		}
	}

	if len(pod.Status.Conditions) > 0 {
		fmt.Println("\nConditions:")
		for _, condition := range pod.Status.Conditions {
			fmt.Printf("  %s: %s\n", condition.Type, condition.Status)
		}
	}

	if len(pod.Status.ContainerStatuses) > 0 {
		fmt.Println("\nContainer Statuses:")
		for _, cs := range pod.Status.ContainerStatuses {
			state := "Unknown"
			if cs.State.Running != nil {
				state = "Running"
			} else if cs.State.Waiting != nil {
				state = fmt.Sprintf("Waiting (%s)", cs.State.Waiting.Reason)
			} else if cs.State.Terminated != nil {
				state = fmt.Sprintf("Terminated (%s)", cs.State.Terminated.Reason)
			}
			fmt.Printf("  %s: %s (restarts: %d)\n", cs.Name, state, cs.RestartCount)
		}
	}

	// Print events if available
	fmt.Println("\nFor events, use: kubectl describe pod", pod.Name, "-n", pod.Namespace)
}

// Helper to get pod status color (for future terminal color support)
func getStatusEmoji(status string) string {
	status = strings.ToLower(status)
	switch status {
	case "running":
		return "✅"
	case "pending":
		return "⏳"
	case "failed":
		return "❌"
	case "succeeded":
		return "✅"
	case "terminating":
		return "🔄"
	default:
		return "❓"
	}
}
