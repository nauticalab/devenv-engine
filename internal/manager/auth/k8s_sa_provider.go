package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/nauticalab/devenv-engine/internal/k8s"
	authv1 "k8s.io/api/authentication/v1"
)

const (
	// DefaultTokenPath is the default path for the projected service account token
	// 	DefaultTokenPath = "/var/run/secrets/tokens/devenv-manager"
	DefaultTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// K8sSAProvider implements authentication via Kubernetes service account tokens
type K8sSAProvider struct {
	// client is the Kubernetes client used for TokenReview (server-side)
	client *k8s.Client
	// audience is the expected audience for tokens (e.g., "devenv-manager")
	audience string
	// namePattern is the pattern to extract developer name (e.g., "devenv-{developer}")
	namePattern string
	// tokenPath is the path to the service account token (client-side)
	tokenPath string
}

// NewK8sSAProvider creates a new Kubernetes service account authentication provider
func NewK8sSAProvider(client *k8s.Client, audience, namePattern, tokenPath string) *K8sSAProvider {
	if namePattern == "" {
		namePattern = "devenv-{developer}"
	}
	if audience == "" {
		audience = "devenv-manager"
	}
	if tokenPath == "" {
		tokenPath = DefaultTokenPath
	}

	return &K8sSAProvider{
		client:      client,
		audience:    audience,
		namePattern: namePattern,
		tokenPath:   tokenPath,
	}
}

// InjectAuth reads the token from the file and adds it to the Authorization header.
func (p *K8sSAProvider) InjectAuth(ctx context.Context, req *http.Request) error {
	tokenBytes, err := os.ReadFile(p.tokenPath)
	if err != nil {
		return fmt.Errorf("failed to read token from %s: %w", p.tokenPath, err)
	}
	token := string(tokenBytes)

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Auth-Type", p.Type())
	return nil
}

// Authenticate validates a Kubernetes service account token using TokenReview API
func (p *K8sSAProvider) Authenticate(ctx context.Context, req *http.Request) (*Identity, error) {
	// Extract token from Authorization header
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return nil, NewAuthError(p.Type(), "missing authorization header", nil)
	}

	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return nil, NewAuthError(p.Type(), "invalid authorization header format", nil)
	}

	token := authHeader[7:]
	if token == "" {
		return nil, NewAuthError(p.Type(), "empty token", nil)
	}

	// Create TokenReview request
	tr := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token:     token,
			Audiences: []string{p.audience},
		},
	}

	// Call Kubernetes TokenReview API
	result, err := p.client.ValidateToken(ctx, tr)
	if err != nil {
		return nil, NewAuthError(p.Type(), "token validation failed", err)
	}

	// Check if token is authenticated
	if !result.Status.Authenticated {
		return nil, NewAuthError(p.Type(), "token not authenticated", nil)
	}

	// Check if token has correct audience
	if len(result.Status.Audiences) > 0 {
		hasCorrectAudience := false
		for _, aud := range result.Status.Audiences {
			if aud == p.audience {
				hasCorrectAudience = true
				break
			}
		}
		if !hasCorrectAudience {
			return nil, NewAuthError(p.Type(), fmt.Sprintf("token audience mismatch, expected %q", p.audience), nil)
		}
	}

	// Extract service account name from username
	// Username format: system:serviceaccount:<namespace>:<sa-name>
	username := result.Status.User.Username
	saName, namespace, err := parseServiceAccountUsername(username)
	if err != nil {
		return nil, NewAuthError(p.Type(), "failed to parse service account", err)
	}

	// Extract developer name from service account name
	developer, err := ParseDeveloperFromServiceAccount(saName, p.namePattern)
	if err != nil {
		return nil, NewAuthError(p.Type(), "failed to extract developer name", err)
	}

	// Build identity
	identity := &Identity{
		Type:       p.Type(),
		Username:   username,
		Developer:  developer,
		Namespace:  namespace,
		Attributes: make(map[string]string),
	}

	// Store additional attributes
	identity.Attributes["sa_name"] = saName
	identity.Attributes["uid"] = result.Status.User.UID

	return identity, nil
}

// Name returns the human-readable name of the provider
func (p *K8sSAProvider) Name() string {
	return "Kubernetes Service Account"
}

// Type returns the provider type identifier
func (p *K8sSAProvider) Type() string {
	return "k8s-sa"
}

// parseServiceAccountUsername parses a Kubernetes service account username
// Format: system:serviceaccount:<namespace>:<sa-name>
// Returns: (sa-name, namespace, error)
func parseServiceAccountUsername(username string) (string, string, error) {
	const prefix = "system:serviceaccount:"

	if len(username) <= len(prefix) {
		return "", "", fmt.Errorf("invalid service account username: %q", username)
	}

	if username[:len(prefix)] != prefix {
		return "", "", fmt.Errorf("not a service account username: %q", username)
	}

	// Remove prefix
	remainder := username[len(prefix):]

	// Split namespace:sa-name
	parts := splitTwo(remainder, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid service account format: %q", username)
	}

	namespace := parts[0]
	saName := parts[1]

	if namespace == "" || saName == "" {
		return "", "", fmt.Errorf("empty namespace or service account name in: %q", username)
	}

	return saName, namespace, nil
}

// splitTwo splits a string on the first occurrence of separator
func splitTwo(s, sep string) []string {
	idx := -1
	for i := 0; i < len(s); i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}
