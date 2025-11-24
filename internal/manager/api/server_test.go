package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nauticalab/devenv-engine/internal/manager/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Start(t *testing.T) {
	// Create a server with a random port
	// config := ServerConfig{
	// 	Port:      0, // Random port
	// 	Audience:  "test-audience",
	// 	K8sClient: nil, // Mock if needed, but Start() mostly sets up http.Server
	// }

	// We can't easily test Start() because it blocks.
	// StartWithContext is better.
}

func TestServer_StartWithContext(t *testing.T) {
	server, _ := setupTestServer()
	// Override addr to random port
	server.addr = ":0"

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should start and then shut down when context is cancelled
	err := server.StartWithContext(ctx)
	// It might return nil or context.DeadlineExceeded depending on timing,
	// but mostly we just want to ensure it doesn't panic or error immediately.
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		t.Errorf("StartWithContext returned unexpected error: %v", err)
	}
}

func TestServer_MiddlewareChain(t *testing.T) {
	// Verify that middleware is applied
	router := chi.NewRouter()

	// Mock provider
	providers := map[string]auth.AuthProvider{
		"mock": &mockProvider{},
	}

	// Setup middleware first
	setupMiddleware(router, providers)

	// Add a test route
	router.Get("/api/v1/middleware-test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 1. Test Security Headers
	// We need a route that exists to test headers, even if it returns 404, headers should be there?
	// Actually, let's use the test route.
	req := httptest.NewRequest("GET", "/api/v1/middleware-test", nil)
	// Add auth header to pass auth middleware for this check if needed,
	// but we want to check headers on any response.
	// However, auth middleware might reject before headers are set if it's early in chain?
	// setupMiddleware adds security headers *before* auth middleware.

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))

	// 2. Test Auth Middleware (should block protected route without token)
	req = httptest.NewRequest("GET", "/api/v1/middleware-test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRespondHelpers(t *testing.T) {
	w := httptest.NewRecorder()
	respondBadRequest(w, "bad request")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = httptest.NewRecorder()
	respondUnauthorized(w, "unauthorized")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = httptest.NewRecorder()
	respondForbidden(w, "forbidden")
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = httptest.NewRecorder()
	respondNotFound(w, "not found")
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = httptest.NewRecorder()
	respondInternalError(w, "internal error")
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	w = httptest.NewRecorder()
	respondCreated(w, map[string]string{"id": "1"})
	assert.Equal(t, http.StatusCreated, w.Code)

	w = httptest.NewRecorder()
	respondNoContent(w)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestNewServer(t *testing.T) {
	config := ServerConfig{
		Port:      8080,
		Audience:  "test",
		K8sClient: nil,
	}
	server, err := NewServer(config)
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, ":8080", server.addr)
}
