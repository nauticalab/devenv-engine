package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nauticalab/devenv-engine/internal/auth"
	"github.com/nauticalab/devenv-engine/internal/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func setupTestServer() (*Server, *k8s.Client) {
	// Create fake K8s client
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dev1-pod",
				Namespace: "default",
				Labels:    map[string]string{"developer": "dev1"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dev2-pod",
				Namespace: "default",
				Labels:    map[string]string{"developer": "dev2"},
			},
		},
	)
	k8sClient := k8s.NewClientWithInterface(clientset)

	// Create handler
	handler := NewHandler(k8sClient, "v1", "commit", "time", "go1.21")

	// Create router
	router := chi.NewRouter()

	// Setup routes directly (skip middleware for handler testing)
	setupRoutes(router, handler)

	return &Server{
		router:  router,
		handler: handler,
	}, k8sClient
}

func TestHandler_ListPods_Security(t *testing.T) {
	server, _ := setupTestServer()

	// Create request with dev1 identity
	req := httptest.NewRequest("GET", "/api/v1/pods", nil)
	ctx := auth.WithIdentity(req.Context(), &auth.Identity{
		Developer: "dev1",
		Username:  "dev1@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp ListPodsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	// Should only see dev1's pod
	assert.Len(t, resp.Pods, 1)
	assert.Equal(t, "dev1-pod", resp.Pods[0].Name)
}

func TestHandler_DeletePod_Security(t *testing.T) {
	server, client := setupTestServer()

	// Try to delete dev2's pod as dev1
	req := httptest.NewRequest("DELETE", "/api/v1/pods/default/dev2-pod", nil)
	ctx := auth.WithIdentity(req.Context(), &auth.Identity{
		Developer: "dev1",
		Username:  "dev1@example.com",
	})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Should fail (either 404 because list filtered it out, or 403 if we had explicit check)
	// In current implementation, ListPodsWithLabels filters it out, so DeletePod won't find it
	// However, DeletePod implementation in handlers.go first checks ownership
	// Let's verify the pod still exists
	pods, _ := client.ListPods(context.Background(), "default")
	assert.Len(t, pods.Items, 2)
}

func TestServer_AuthMiddleware(t *testing.T) {
	// This tests the middleware logic specifically
	router := chi.NewRouter()

	// Mock provider
	providers := map[string]auth.AuthProvider{
		"mock": &mockProvider{},
	}

	setupMiddleware(router, providers)

	// Add a test route
	router.Get("/api/v1/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test public endpoint
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusUnauthorized, w.Code) // Should pass auth

	// Test protected endpoint without token
	req = httptest.NewRequest("GET", "/api/v1/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

type mockProvider struct{}

func (m *mockProvider) Authenticate(ctx context.Context, token string) (*auth.Identity, error) {
	return nil, nil
}

func (m *mockProvider) Name() string {
	return "Mock Provider"
}

func (m *mockProvider) Type() string {
	return "mock"
}
