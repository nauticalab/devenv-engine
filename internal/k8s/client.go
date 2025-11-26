// Package k8s provides a wrapper around the Kubernetes client-go library.
// It simplifies common operations such as listing pods, authenticating tokens,
// and managing Kubernetes resources for the DevEnv engine.
package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the Kubernetes clientset and provides helper methods
type Client struct {
	// clientset is the standard Kubernetes clientset interface
	clientset kubernetes.Interface
}

// NewClientWithInterface creates a new Kubernetes client with the provided interface.
// This is primarily used for testing with fake clientsets.
func NewClientWithInterface(clientset kubernetes.Interface) *Client {
	return &Client{clientset: clientset}
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
// If namespace is empty, it lists pods across all namespaces.
func (c *Client) ListPods(ctx context.Context, namespace string) (*corev1.PodList, error) {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	return pods, nil
}

// ListPodsWithLabels lists pods in the specified namespace matching the given label selector.
// If namespace is empty, it lists pods across all namespaces.
// Example labelSelector: "app=devenv,developer=eywalker"
func (c *Client) ListPodsWithLabels(ctx context.Context, namespace, labelSelector string) (*corev1.PodList, error) {
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

// DeletePod deletes a pod by namespace and name
func (c *Client) DeletePod(ctx context.Context, namespace, name string) error {
	err := c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod %s/%s: %w", namespace, name, err)
	}
	return nil
}

// GetPodByName retrieves a specific pod by namespace and name
func (c *Client) GetPodByName(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}
	return pod, nil
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

// ValidateToken validates a service account token using the TokenReview API
func (c *Client) ValidateToken(ctx context.Context, tokenReview *authv1.TokenReview) (*authv1.TokenReview, error) {
	result, err := c.clientset.AuthenticationV1().TokenReviews().Create(ctx, tokenReview, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create TokenReview: %w", err)
	}
	return result, nil
}
