package auth

import (
	"context"
	"net/http"
)

// AuthProvider defines the interface for authentication providers
// Different providers (K8s SA, GitHub, OIDC) implement this interface
type AuthProvider interface {
	// Authenticate validates the request and returns an Identity.
	// It returns an error if authentication fails.
	Authenticate(ctx context.Context, req *http.Request) (*Identity, error)

	// InjectAuth adds authentication information (e.g., headers) to the request.
	// This is used by the client to inject credentials.
	InjectAuth(ctx context.Context, req *http.Request) error

	// Name returns a human-readable name for the provider (e.g., "Kubernetes Service Account")
	Name() string

	// Type returns the provider type identifier (e.g., "k8s-sa", "github")
	Type() string
}

// AuthError represents an authentication error
type AuthError struct {
	Provider string
	Reason   string
	Err      error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return e.Provider + ": " + e.Reason + ": " + e.Err.Error()
	}
	return e.Provider + ": " + e.Reason
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// NewAuthError creates a new authentication error
func NewAuthError(provider, reason string, err error) *AuthError {
	return &AuthError{
		Provider: provider,
		Reason:   reason,
		Err:      err,
	}
}
