// Package k8s provides a wrapper around the Kubernetes client-go library.
// It simplifies common operations such as listing pods, authenticating tokens,
// and managing Kubernetes resources for the DevEnv engine.
package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
