package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nauticalab/devenv-engine/internal/auth"
	"github.com/nauticalab/devenv-engine/internal/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHandler_Health(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	server.handler.Health(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.False(t, resp.Timestamp.IsZero())
}

func TestHandler_Version(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/v1/version", nil)
	w := httptest.NewRecorder()

	server.handler.Version(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp VersionResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "v1", resp.Version)
	assert.Equal(t, "commit", resp.GitCommit)
	assert.Equal(t, "time", resp.BuildTime)
	assert.Equal(t, "go1.21", resp.GoVersion)
}

func TestHandler_ListPods(t *testing.T) {
	server, _ := setupTestServer()

	// Test with dev1
	req := httptest.NewRequest("GET", "/api/v1/pods", nil)
	ctx := auth.WithIdentity(req.Context(), &auth.Identity{
		Developer: "dev1",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	server.handler.ListPods(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp ListPodsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Pods, 1)
	assert.Equal(t, "dev1-pod", resp.Pods[0].Name)

	// Test with dev2
	req = httptest.NewRequest("GET", "/api/v1/pods", nil)
	ctx = auth.WithIdentity(req.Context(), &auth.Identity{
		Developer: "dev2",
	})
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()

	server.handler.ListPods(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Pods, 1)
	assert.Equal(t, "dev2-pod", resp.Pods[0].Name)

	// Test with no identity (should fail)
	req = httptest.NewRequest("GET", "/api/v1/pods", nil)
	w = httptest.NewRecorder()

	server.handler.ListPods(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_DeletePod(t *testing.T) {
	server, client := setupTestServer()

	// Test successful deletion
	req := httptest.NewRequest("DELETE", "/api/v1/pods/default/dev1-pod", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("namespace", "default")
	rctx.URLParams.Add("name", "dev1-pod")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	ctx := auth.WithIdentity(req.Context(), &auth.Identity{
		Developer: "dev1",
	})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	server.handler.DeletePod(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp DeletePodResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// Verify deletion
	pods, _ := client.ListPods(context.Background(), "default")
	assert.Len(t, pods.Items, 1) // dev2-pod remains

	// Test deleting other's pod
	req = httptest.NewRequest("DELETE", "/api/v1/pods/default/dev2-pod", nil)
	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("namespace", "default")
	rctx.URLParams.Add("name", "dev2-pod")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	ctx = auth.WithIdentity(req.Context(), &auth.Identity{
		Developer: "dev1",
	})
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()

	server.handler.DeletePod(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_WhoAmI(t *testing.T) {
	// Setup
	k8sClient := &k8s.Client{} // Mock client not needed for this test
	handler := NewHandler(k8sClient, "1.0.0", "abcdef", "now", "1.21")

	// Create request with identity
	req := httptest.NewRequest("GET", "/api/v1/auth/whoami", nil)
	identity := &auth.Identity{
		Type:      "k8s-sa",
		Username:  "system:serviceaccount:default:devenv-eywalker",
		Developer: "eywalker",
	}
	ctx := auth.WithIdentity(req.Context(), identity)
	req = req.WithContext(ctx)

	// Create recorder
	w := httptest.NewRecorder()

	// Execute
	handler.WhoAmI(w, req)

	// Verify
	assert.Equal(t, http.StatusOK, w.Code)

	var response WhoAmIResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "eywalker", response.Developer)
	assert.Equal(t, "k8s-sa", response.Type)
}

func TestHandler_WhoAmI_Unauthenticated(t *testing.T) {
	// Setup
	k8sClient := &k8s.Client{}
	handler := NewHandler(k8sClient, "1.0.0", "abcdef", "now", "1.21")

	// Create request without identity
	req := httptest.NewRequest("GET", "/api/v1/auth/whoami", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.WhoAmI(w, req)

	// Verify
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
