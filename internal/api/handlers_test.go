package api

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertPodToAPI(t *testing.T) {
	now := time.Now()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "test-ns",
			CreationTimestamp: metav1.NewTime(now.Add(-5 * time.Minute)),
			Labels: map[string]string{
				"developer": "testuser",
				"app":       "devenv",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
			ContainerStatuses: []corev1.ContainerStatus{
				{RestartCount: 2},
				{RestartCount: 1},
			},
		},
	}

	result := convertPodToAPI(pod)

	if result.Name != "test-pod" {
		t.Errorf("Name = %v, want test-pod", result.Name)
	}

	if result.Namespace != "test-ns" {
		t.Errorf("Namespace = %v, want test-ns", result.Namespace)
	}

	if result.Status != "Running" {
		t.Errorf("Status = %v, want Running", result.Status)
	}

	if result.Restarts != 3 {
		t.Errorf("Restarts = %v, want 3", result.Restarts)
	}

	if result.Developer != "testuser" {
		t.Errorf("Developer = %v, want testuser", result.Developer)
	}

	if result.NodeName != "node-1" {
		t.Errorf("NodeName = %v, want node-1", result.NodeName)
	}

	if result.PodIP != "10.0.0.1" {
		t.Errorf("PodIP = %v, want 10.0.0.1", result.PodIP)
	}

	// Age should be in format "5m"
	if result.Age != "5m" {
		t.Errorf("Age = %v, want 5m", result.Age)
	}
}

func TestConvertPodToAPI_NoDeveloperLabel(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "test-ns",
			CreationTimestamp: metav1.Now(),
			Labels:            map[string]string{},
		},
		Spec:   corev1.PodSpec{},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	result := convertPodToAPI(pod)

	if result.Developer != "-" {
		t.Errorf("Developer = %v, want -", result.Developer)
	}
}

func TestConvertPodToAPI_Terminating(t *testing.T) {
	now := metav1.Now()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "test-ns",
			CreationTimestamp: metav1.Now(),
			DeletionTimestamp: &now,
		},
		Spec:   corev1.PodSpec{},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	result := convertPodToAPI(pod)

	if result.Status != "Terminating" {
		t.Errorf("Status = %v, want Terminating", result.Status)
	}
}

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		time    time.Time
		wantAge string
	}{
		{
			name:    "seconds",
			time:    now.Add(-30 * time.Second),
			wantAge: "30s",
		},
		{
			name:    "minutes",
			time:    now.Add(-5 * time.Minute),
			wantAge: "5m",
		},
		{
			name:    "hours",
			time:    now.Add(-3 * time.Hour),
			wantAge: "3h",
		},
		{
			name:    "days",
			time:    now.Add(-2 * 24 * time.Hour),
			wantAge: "2d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(tt.time)
			if got != tt.wantAge {
				t.Errorf("formatAge() = %v, want %v", got, tt.wantAge)
			}
		})
	}
}
