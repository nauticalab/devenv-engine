package manager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nauticalab/devenv-engine/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListPods(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/pods", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		// Return mock response
		resp := api.ListPodsResponse{
			Pods: []api.Pod{
				{Name: "pod-1", Namespace: "default", Status: "Running"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	tmpToken := createTempToken(t, "test-token")
	defer os.Remove(tmpToken)

	client := NewClient(ClientConfig{
		BaseURL:   server.URL,
		TokenPath: tmpToken,
	})

	// Test list pods
	resp, err := client.ListPods(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, resp.Pods, 1)
	assert.Equal(t, "pod-1", resp.Pods[0].Name)
}

func TestClient_DeletePod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/v1/pods/default/test-pod", r.URL.Path)

		resp := api.DeletePodResponse{
			Success: true,
			Message: "Pod deleted",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpToken := createTempToken(t, "test-token")
	defer os.Remove(tmpToken)

	client := NewClient(ClientConfig{
		BaseURL:   server.URL,
		TokenPath: tmpToken,
	})

	resp, err := client.DeletePod(context.Background(), "default", "test-pod")
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestClient_Health(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/health", r.URL.Path)

		resp := api.HealthResponse{
			Status:    "ok",
			Timestamp: time.Now(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpToken := createTempToken(t, "test-token")
	defer os.Remove(tmpToken)

	client := NewClient(ClientConfig{
		BaseURL:   server.URL,
		TokenPath: tmpToken,
	})

	resp, err := client.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestClient_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := api.ErrorResponse{
			Code:    400,
			Message: "Bad Request",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpToken := createTempToken(t, "test-token")
	defer os.Remove(tmpToken)

	client := NewClient(ClientConfig{
		BaseURL:   server.URL,
		TokenPath: tmpToken,
	})

	_, err := client.ListPods(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error: Bad Request")
}

func createTempToken(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "token")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()
	return tmpFile.Name()
}

func TestClient_Version(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/version", r.URL.Path)

		resp := api.VersionResponse{
			Version:   "v1.0.0",
			GitCommit: "abcdef",
			BuildTime: "2023-01-01",
			GoVersion: "go1.21",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tmpToken := createTempToken(t, "test-token")
	defer os.Remove(tmpToken)

	client := NewClient(ClientConfig{
		BaseURL:   server.URL,
		TokenPath: tmpToken,
	})

	resp, err := client.Version(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", resp.Version)
	assert.Equal(t, "abcdef", resp.GitCommit)
}

func TestNewClient_Defaults(t *testing.T) {
	client := NewClient(ClientConfig{
		BaseURL: "http://example.com",
	})

	assert.Equal(t, "http://example.com", client.baseURL)
	assert.Equal(t, DefaultTokenPath, client.tokenPath)
	assert.Equal(t, DefaultTimeout, client.httpClient.Timeout)
	assert.Equal(t, "k8s-sa", client.authType)
}

func TestClient_ReadTokenError(t *testing.T) {
	client := NewClient(ClientConfig{
		BaseURL:   "http://example.com",
		TokenPath: "/non/existent/path",
	})

	_, err := client.ListPods(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read token")
}

func TestClient_ParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid-json"))
	}))
	defer server.Close()

	tmpToken := createTempToken(t, "test-token")
	defer os.Remove(tmpToken)

	client := NewClient(ClientConfig{
		BaseURL:   server.URL,
		TokenPath: tmpToken,
	})

	_, err := client.ListPods(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestClient_ParseErrorResponse_Malformed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid-json-error"))
	}))
	defer server.Close()

	tmpToken := createTempToken(t, "test-token")
	defer os.Remove(tmpToken)

	client := NewClient(ClientConfig{
		BaseURL:   server.URL,
		TokenPath: tmpToken,
	})

	_, err := client.ListPods(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
}
