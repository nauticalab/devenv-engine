package manager

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nauticalab/devenv-engine/internal/k8s"
	corev1 "k8s.io/api/core/v1"
)

// PodsOptions holds configuration for the manager pods command
type PodsOptions struct {
	Namespace     string
	AllNamespaces bool
	LabelFilter   string
	ShowAll       bool
	Verbose       bool
}

// RunListPods lists running pods
func RunListPods(developerName string, opts PodsOptions) {
	ctx := context.Background()

	// Create Kubernetes client
	client, err := k8s.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure you have kubectl configured and can access the cluster.\n")
		os.Exit(1)
	}

	var pods *corev1.PodList

	// Determine how to list pods
	if opts.AllNamespaces {
		if opts.Verbose {
			fmt.Println("Listing pods across all namespaces...")
		}
		pods, err = client.ListAllPods(ctx)
	} else if developerName != "" {
		// Filter by developer name using label selector
		labelSelector := fmt.Sprintf("developer=%s", developerName)
		if opts.LabelFilter != "" {
			labelSelector = fmt.Sprintf("%s,%s", labelSelector, opts.LabelFilter)
		}
		if opts.Verbose {
			fmt.Printf("Listing pods for developer: %s (label: %s)\n", developerName, labelSelector)
		}
		pods, err = client.ListPodsWithLabels(ctx, opts.Namespace, labelSelector)
	} else if opts.LabelFilter != "" {
		if opts.Verbose {
			fmt.Printf("Listing pods with labels: %s\n", opts.LabelFilter)
		}
		pods, err = client.ListPodsWithLabels(ctx, opts.Namespace, opts.LabelFilter)
	} else {
		if opts.Verbose {
			fmt.Printf("Listing pods in namespace: %s\n", opts.Namespace)
		}
		pods, err = client.ListPods(ctx, opts.Namespace)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing pods: %v\n", err)
		os.Exit(1)
	}

	// Filter pods if needed
	var filteredPods []corev1.Pod
	for _, pod := range pods.Items {
		if opts.ShowAll || k8s.IsPodRunning(&pod) {
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
	printMgrPodsTable(filteredPods, opts.AllNamespaces)
}

func printMgrPodsTable(pods []corev1.Pod, allNamespaces bool) {
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
		status := k8s.GetPodStatus(&pod)
		restarts := getMgrTotalRestarts(&pod)
		age := formatMgrAge(pod.CreationTimestamp.Time)
		developer := pod.Labels["developer"]
		if developer == "" {
			developer = "-"
		}

		if allNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				pod.Namespace, pod.Name, status, restarts, age, developer)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				pod.Name, status, restarts, age, developer)
		}
	}
}

// getMgrTotalRestarts calculates the total number of container restarts in a pod
func getMgrTotalRestarts(pod *corev1.Pod) int32 {
	var total int32
	for _, cs := range pod.Status.ContainerStatuses {
		total += cs.RestartCount
	}
	return total
}

// formatMgrAge returns a human-readable duration string
func formatMgrAge(t time.Time) string {
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

// Helper to get pod status color (for future terminal color support)
func getMgrStatusEmoji(status string) string {
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
