package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/nauticalab/devenv-engine/internal/api"
)

const (
	// DefaultTokenPath is the default path for the projected service account token
	DefaultTokenPath = "/var/run/secrets/tokens/devenv-manager"

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
)

// Client represents an HTTP client for the DevEnv Manager API
type Client struct {
	baseURL    string
	httpClient *http.Client
	tokenPath  string
	authType   string
}

// ClientConfig holds configuration for the manager client
type ClientConfig struct {
	BaseURL   string
	TokenPath string
	Timeout   time.Duration
	AuthType  string
}

// NewClient creates a new manager API client
func NewClient(config ClientConfig) *Client {
	if config.TokenPath == "" {
		config.TokenPath = DefaultTokenPath
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}
	if config.AuthType == "" {
		config.AuthType = "k8s-sa"
	}

	return &Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		tokenPath: config.TokenPath,
		authType:  config.AuthType,
	}
}

// readToken reads the service account token from the mounted volume
func (c *Client) readToken() (string, error) {
	tokenBytes, err := os.ReadFile(c.tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read token from %s: %w", c.tokenPath, err)
	}
	return string(tokenBytes), nil
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	// Read the service account token
	token, err := c.readToken()
	if err != nil {
		return nil, err
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Create request
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Auth-Type", c.authType)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Perform request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// parseResponse parses the HTTP response into the target structure
func parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		var errResp api.ErrorResponse
		if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
		}
		return fmt.Errorf("API error: %s (code: %d)", errResp.Message, errResp.Code)
	}

	// Parse success response
	if target != nil {
		if err := json.Unmarshal(bodyBytes, target); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// Health checks the health of the manager API server
func (c *Client) Health(ctx context.Context) (*api.HealthResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/health", nil)
	if err != nil {
		return nil, err
	}

	var health api.HealthResponse
	if err := parseResponse(resp, &health); err != nil {
		return nil, err
	}

	return &health, nil
}

// Version retrieves version information from the manager API server
func (c *Client) Version(ctx context.Context) (*api.VersionResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/version", nil)
	if err != nil {
		return nil, err
	}

	var version api.VersionResponse
	if err := parseResponse(resp, &version); err != nil {
		return nil, err
	}

	return &version, nil
}

// ListPods retrieves the list of pods for the authenticated developer
func (c *Client) ListPods(ctx context.Context, namespace string) (*api.ListPodsResponse, error) {
	path := "/api/v1/pods"
	if namespace != "" {
		path = fmt.Sprintf("%s?namespace=%s", path, namespace)
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var listResp api.ListPodsResponse
	if err := parseResponse(resp, &listResp); err != nil {
		return nil, err
	}

	return &listResp, nil
}

// DeletePod deletes a pod by namespace and name
func (c *Client) DeletePod(ctx context.Context, namespace, name string) (*api.DeletePodResponse, error) {
	path := fmt.Sprintf("/api/v1/pods/%s/%s", namespace, name)

	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	var deleteResp api.DeletePodResponse
	if err := parseResponse(resp, &deleteResp); err != nil {
		return nil, err
	}

	return &deleteResp, nil
}
