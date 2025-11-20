package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes clientset and provides helper methods
type Client struct {
	clientset *kubernetes.Clientset
}

// NewClient creates a new Kubernetes client using the standard kubeconfig location
// or in-cluster config if running inside a Kubernetes cluster.
func NewClient() (*Client, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &Client{clientset: clientset}, nil
}

// getKubeConfig attempts to load Kubernetes configuration from the following sources in order:
// 1. In-cluster config (if running inside a pod)
// 2. KUBECONFIG environment variable
// 3. ~/.kube/config (default kubeconfig location)
func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// Use default kubeconfig location
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

// ListPods lists all pods in the specified namespace.
// If namespace is empty, it defaults to "default".
func (c *Client) ListPods(ctx context.Context, namespace string) (*corev1.PodList, error) {
	if namespace == "" {
		namespace = "default"
	}

	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	return pods, nil
}

// ListPodsWithLabels lists pods in the specified namespace matching the given label selector.
// If namespace is empty, it defaults to "default".
// Example labelSelector: "app=devenv,developer=eywalker"
func (c *Client) ListPodsWithLabels(ctx context.Context, namespace, labelSelector string) (*corev1.PodList, error) {
	if namespace == "" {
		namespace = "default"
	}

	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods with labels %s in namespace %s: %w", labelSelector, namespace, err)
	}

	return pods, nil
}

// ListAllPods lists all pods across all namespaces
func (c *Client) ListAllPods(ctx context.Context) (*corev1.PodList, error) {
	pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list all pods: %w", err)
	}

	return pods, nil
}

// GetPodStatus returns a simplified status string for a pod
func GetPodStatus(pod *corev1.Pod) string {
	// Check if pod is being deleted
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}

	// Return the pod phase
	return string(pod.Status.Phase)
}

// IsPodRunning returns true if the pod is in Running phase
func IsPodRunning(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodRunning
}
