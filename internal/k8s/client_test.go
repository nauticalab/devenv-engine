package k8s

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestClient_ListPods(t *testing.T) {
	// Create fake clientset with some pods
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "test"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-2",
				Namespace: "other",
			},
		},
	)

	client := NewClientWithInterface(clientset)

	// Test listing in default namespace
	pods, err := client.ListPods(context.Background(), "default")
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "pod-1", pods.Items[0].Name)

	// Test listing in other namespace
	pods, err = client.ListPods(context.Background(), "other")
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "pod-2", pods.Items[0].Name)

	// Test listing in empty namespace (lists all namespaces)
	pods, err = client.ListPods(context.Background(), "")
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 2)
}

func TestClient_ListAllPods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "ns1",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-2",
				Namespace: "ns2",
			},
		},
	)

	client := NewClientWithInterface(clientset)

	pods, err := client.ListAllPods(context.Background())
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 2)
}

func TestClient_ListPodsWithLabels(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "app1", "env": "prod"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-2",
				Namespace: "default",
				Labels:    map[string]string{"app": "app2", "env": "prod"},
			},
		},
	)

	client := NewClientWithInterface(clientset)

	// Test matching label
	pods, err := client.ListPodsWithLabels(context.Background(), "default", "app=app1")
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "pod-1", pods.Items[0].Name)

	// Test matching multiple labels
	pods, err = client.ListPodsWithLabels(context.Background(), "default", "env=prod")
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 2)

	// Test no match
	pods, err = client.ListPodsWithLabels(context.Background(), "default", "app=nonexistent")
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 0)
}

func TestClient_ListPodsWithLabels_AllNamespaces(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "test"},
			},
		},
	)

	client := NewClientWithInterface(clientset)

	// Test with empty namespace (lists all namespaces)
	pods, err := client.ListPodsWithLabels(context.Background(), "", "app=test")
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "pod-1", pods.Items[0].Name)
}

func TestClient_DeletePod(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-to-delete",
				Namespace: "default",
			},
		},
	)

	client := NewClientWithInterface(clientset)

	// Verify pod exists
	pods, _ := client.ListPods(context.Background(), "default")
	assert.Len(t, pods.Items, 1)

	// Delete pod
	err := client.DeletePod(context.Background(), "default", "pod-to-delete")
	assert.NoError(t, err)

	// Verify pod is gone
	pods, _ = client.ListPods(context.Background(), "default")
	assert.Len(t, pods.Items, 0)

	// Delete non-existent pod should fail
	err = client.DeletePod(context.Background(), "default", "nonexistent")
	assert.Error(t, err)
}

func TestGetPodStatus(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected string
	}{
		{
			name: "Running",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
			},
			expected: "Running",
		},
		{
			name: "Terminating",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{},
				},
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
			},
			expected: "Terminating",
		},
		{
			name: "Pending",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodPending},
			},
			expected: "Pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetPodStatus(tt.pod))
		})
	}
}

func TestIsPodRunning(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected bool
	}{
		{
			name: "Running",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
			},
			expected: true,
		},
		{
			name: "Pending",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodPending},
			},
			expected: false,
		},
		{
			name: "Failed",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodFailed},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsPodRunning(tt.pod))
		})
	}
}

func TestClient_GetPodByName(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "target-pod",
				Namespace: "default",
			},
		},
	)

	client := NewClientWithInterface(clientset)

	// Test existing pod
	pod, err := client.GetPodByName(context.Background(), "default", "target-pod")
	assert.NoError(t, err)
	assert.Equal(t, "target-pod", pod.Name)

	// Test non-existent pod
	_, err = client.GetPodByName(context.Background(), "default", "missing-pod")
	assert.Error(t, err)
}

func TestClient_ValidateToken(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	client := NewClientWithInterface(clientset)

	tr := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: "test-token",
		},
	}

	// The fake client doesn't actually validate tokens, but it should return a result without error
	result, err := client.ValidateToken(context.Background(), tr)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClient_ListPods_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("simulated error")
	})

	client := NewClientWithInterface(clientset)
	_, err := client.ListPods(context.Background(), "default")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated error")
}

func TestClient_ListPodsWithLabels_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("simulated error")
	})

	client := NewClientWithInterface(clientset)
	_, err := client.ListPodsWithLabels(context.Background(), "default", "app=test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated error")
}

func TestClient_ListAllPods_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("simulated error")
	})

	client := NewClientWithInterface(clientset)
	_, err := client.ListAllPods(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated error")
}

func TestClient_ValidateToken_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, fmt.Errorf("simulated error")
	})

	client := NewClientWithInterface(clientset)
	tr := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: "test-token",
		},
	}
	_, err := client.ValidateToken(context.Background(), tr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated error")
}
