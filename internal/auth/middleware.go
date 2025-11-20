package auth

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const (
	// AuthTypeHeader is the HTTP header for specifying auth provider type
	AuthTypeHeader = "X-Auth-Type"

	// DefaultAuthType is used when X-Auth-Type header is not provided
	DefaultAuthType = "k8s-sa"
)

// Middleware creates an HTTP middleware that validates authentication on every request
// This implements zero-trust: no sessions, validate every request
func Middleware(providers map[string]AuthProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			token, err := extractBearerToken(r)
			if err != nil {
				log.Printf("Auth failed: %v", err)
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Get requested auth type from header, default to k8s-sa
			authType := r.Header.Get(AuthTypeHeader)
			if authType == "" {
				authType = DefaultAuthType
			}

			// Find the requested provider
			provider, ok := providers[authType]
			if !ok {
				log.Printf("Auth failed: unknown auth type %q", authType)
				http.Error(w, fmt.Sprintf("Unauthorized: unknown auth type %q", authType), http.StatusUnauthorized)
				return
			}

			// Authenticate using the provider
			identity, err := provider.Authenticate(r.Context(), token)
			if err != nil {
				log.Printf("Auth failed with %s provider: %v", authType, err)
				http.Error(w, "Unauthorized: authentication failed", http.StatusUnauthorized)
				return
			}

			// Store identity in context
			ctx := WithIdentity(r.Context(), identity)

			// Log successful authentication
			log.Printf("Authenticated: %s (developer=%s)", identity.Username, identity.Developer)

			// Call next handler with authenticated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractBearerToken extracts the bearer token from the Authorization header
// Expected format: "Authorization: Bearer <token>"
func extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	if strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("authorization scheme must be Bearer")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}

	return token, nil
}

// RequireAuthentication is a helper that returns 401 if no identity in context
// Use this in handlers that require authentication
func RequireAuthentication(w http.ResponseWriter, r *http.Request) (*Identity, bool) {
	identity, ok := GetIdentityFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return identity, true
}

// RequireDeveloper is a helper that extracts and validates developer from context
func RequireDeveloper(w http.ResponseWriter, r *http.Request) (string, bool) {
	developer, ok := GetDeveloperFromContext(r.Context())
	if !ok || developer == "" {
		http.Error(w, "Forbidden: no developer identity", http.StatusForbidden)
		return "", false
	}
	return developer, true
}
