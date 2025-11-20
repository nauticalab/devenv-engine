package auth

import (
	"context"
	"fmt"
	"strings"
)

// Identity represents an authenticated user/service account
type Identity struct {
	// Type of authentication (e.g., "k8s-sa", "github", "oidc")
	Type string

	// Username is the full identifier (e.g., service account name, GitHub username)
	Username string

	// Developer is the resolved developer name extracted from the identity
	Developer string

	// Namespace is the Kubernetes namespace (for k8s-sa type)
	Namespace string

	// Attributes holds additional metadata/claims
	Attributes map[string]string
}

// String returns a human-readable representation of the identity
func (i *Identity) String() string {
	return fmt.Sprintf("%s:%s (developer=%s)", i.Type, i.Username, i.Developer)
}

// contextKey is used for storing identity in request context
type contextKey string

const (
	// IdentityContextKey is the key for storing Identity in context
	IdentityContextKey contextKey = "identity"

	// DeveloperContextKey is the key for storing developer name in context
	DeveloperContextKey contextKey = "developer"
)

// GetIdentityFromContext extracts the Identity from the request context
func GetIdentityFromContext(ctx context.Context) (*Identity, bool) {
	identity, ok := ctx.Value(IdentityContextKey).(*Identity)
	return identity, ok
}

// GetDeveloperFromContext extracts the developer name from the request context
func GetDeveloperFromContext(ctx context.Context) (string, bool) {
	developer, ok := ctx.Value(DeveloperContextKey).(string)
	return developer, ok
}

// WithIdentity returns a new context with the identity stored
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	ctx = context.WithValue(ctx, IdentityContextKey, identity)
	ctx = context.WithValue(ctx, DeveloperContextKey, identity.Developer)
	return ctx
}

// ParseDeveloperFromServiceAccount extracts the developer name from a service account name
// using the pattern "devenv-{developer}"
func ParseDeveloperFromServiceAccount(saName, pattern string) (string, error) {
	// Default pattern: "devenv-{developer}"
	if pattern == "" {
		pattern = "devenv-{developer}"
	}

	// For now, support simple prefix pattern
	// Future: support more complex patterns with regex
	prefix := strings.Split(pattern, "{developer}")[0]

	if !strings.HasPrefix(saName, prefix) {
		return "", fmt.Errorf("service account name %q does not match pattern %q", saName, pattern)
	}

	developer := strings.TrimPrefix(saName, prefix)
	if developer == "" {
		return "", fmt.Errorf("could not extract developer name from %q using pattern %q", saName, pattern)
	}

	return developer, nil
}
