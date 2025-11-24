// Package client provides a client for interacting with the DevENV Manager API.
// It handles authentication, request execution, and response parsing.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nauticalab/devenv-engine/internal/manager/api"
	"github.com/nauticalab/devenv-engine/internal/manager/auth"
)

const (
	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
)

// Client represents an HTTP client for the DevENV Manager API
type Client struct {
	// baseURL is the base URL of the manager API
	baseURL string
	// httpClient is the underlying HTTP client
	httpClient *http.Client
	// authProvider is the authentication provider
	authProvider auth.AuthProvider
}

// NewClient creates a new manager API client
func NewClient(baseURL string, authProvider auth.AuthProvider) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		authProvider: authProvider,
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
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

	// Inject authentication
	if c.authProvider != nil {
		if err := c.authProvider.InjectAuth(ctx, req); err != nil {
			return nil, fmt.Errorf("failed to inject authentication: %w", err)
		}
	}

	// Add headers
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
func parseResponse(resp *http.Response, target any) error {
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

// WhoAmI retrieves the identity of the authenticated user
func (c *Client) WhoAmI(ctx context.Context) (*api.WhoAmIResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/auth/whoami", nil)
	if err != nil {
		return nil, err
	}

	var whoami api.WhoAmIResponse
	if err := parseResponse(resp, &whoami); err != nil {
		return nil, err
	}

	return &whoami, nil
}

// ListPods retrieves the list of pods for the authenticated developer
func (c *Client) ListPods(ctx context.Context, namespace string, allNamespaces bool) (*api.ListPodsResponse, error) {
	path := "/api/v1/pods"
	queryParams := []string{}

	if namespace != "" {
		queryParams = append(queryParams, fmt.Sprintf("namespace=%s", namespace))
	}
	if allNamespaces {
		queryParams = append(queryParams, "all_namespaces=true")
	}

	if len(queryParams) > 0 {
		path = fmt.Sprintf("%s?%s", path, strings.Join(queryParams, "&"))
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
