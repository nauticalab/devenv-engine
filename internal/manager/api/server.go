package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/nauticalab/devenv-engine/internal/k8s"
	"github.com/nauticalab/devenv-engine/internal/manager/auth"
)

// Server represents the HTTP API server
type Server struct {
	// router is the HTTP request multiplexer
	router *chi.Mux
	// handler contains the API route handlers
	handler *Handler
	// providers is a map of registered authentication providers
	providers map[string]auth.AuthProvider
	// addr is the address the server listens on
	addr string
	// tlsCertPath is the path to the TLS certificate file
	tlsCertPath string
	// tlsKeyPath is the path to the TLS private key file
	tlsKeyPath string
}

// ServerConfig holds configuration for the API server
type ServerConfig struct {
	// Port is the TCP port to listen on
	Port int
	// Audience is the expected audience for JWT tokens
	Audience string
	// K8sClient is the Kubernetes client
	K8sClient *k8s.Client
	// Version is the application version
	Version string
	// GitCommit is the git commit hash
	GitCommit string
	// BuildTime is the build timestamp
	BuildTime string
	// GoVersion is the Go version used for the build
	GoVersion string
	// TLSCertPath is the path to the TLS certificate file
	TLSCertPath string
	// TLSKeyPath is the path to the TLS private key file
	TLSKeyPath string
}

// NewServer creates a new API server with the given configuration
func NewServer(config ServerConfig) (*Server, error) {
	// Create K8s SA auth provider
	k8sProvider := auth.NewK8sSAProvider(config.K8sClient, config.Audience, "devenv-{developer}", "")

	// Build provider map
	providers := map[string]auth.AuthProvider{
		"k8s-sa": k8sProvider,
	}

	// Create handler
	handler := NewHandler(
		config.K8sClient,
		config.Version,
		config.GitCommit,
		config.BuildTime,
		config.GoVersion,
	)

	// Create router
	router := chi.NewRouter()

	// Setup middleware
	setupMiddleware(router, providers)

	// Setup routes
	setupRoutes(router, handler)

	addr := fmt.Sprintf(":%d", config.Port)

	return &Server{
		router:      router,
		handler:     handler,
		providers:   providers,
		addr:        addr,
		tlsCertPath: config.TLSCertPath,
		tlsKeyPath:  config.TLSKeyPath,
	}, nil
}

// setupMiddleware configures the middleware chain
func setupMiddleware(router *chi.Mux, providers map[string]auth.AuthProvider) {
	// Request logger
	router.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
		Logger:  log.Default(),
		NoColor: false,
	}))

	// Recoverer from panics
	router.Use(middleware.Recoverer)

	// Timeout for requests
	router.Use(middleware.Timeout(60 * time.Second))

	// Security headers
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Content-Security-Policy", "default-src 'none'")
			next.ServeHTTP(w, r)
		})
	})

	// Rate limiting: 100 requests per minute per IP
	router.Use(httprate.LimitByIP(100, 1*time.Minute))

	// Create auth middleware
	authMiddleware := auth.Middleware(providers)

	// Apply authentication globally with exemptions for public endpoints
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exempt public endpoints
			if r.URL.Path == "/api/v1/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Apply authentication to all other endpoints
			authMiddleware(next).ServeHTTP(w, r)
		})
	})
}

// setupRoutes configures the API routes
func setupRoutes(router *chi.Mux, handler *Handler) {
	// API v1 routes
	router.Route("/api/v1", func(r chi.Router) {
		// Public endpoints
		r.Get("/health", handler.Health)
		r.Get("/version", handler.Version)

		// Protected endpoints (auth middleware applied at server level)
		r.Group(func(r chi.Router) {
			// Pods endpoints
			r.Get("/pods", handler.ListPods)
			r.Delete("/pods/{namespace}/{name}", handler.DeletePod)

			// Auth endpoints
			r.Get("/auth/whoami", handler.WhoAmI)
		})
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting API server on %s", s.addr)
	log.Printf("Registered auth providers: %v", s.getProviderNames())

	server := &http.Server{
		Addr:         s.addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Check if TLS is configured
	if s.tlsCertPath != "" && s.tlsKeyPath != "" {
		log.Printf("Starting HTTPS server with TLS")
		return server.ListenAndServeTLS(s.tlsCertPath, s.tlsKeyPath)
	}

	log.Printf("Starting HTTP server (TLS not configured)")
	return server.ListenAndServe()
}

// StartWithContext starts the HTTP server with graceful shutdown support
func (s *Server) StartWithContext(ctx context.Context) error {
	log.Printf("Starting API server on %s", s.addr)
	log.Printf("Registered auth providers: %v", s.getProviderNames())

	server := &http.Server{
		Addr:         s.addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to signal server errors
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on %s", s.addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		log.Println("Shutting down server...")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
			return err
		}

		log.Println("Server stopped gracefully")
		return nil

	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}
}

// getProviderNames returns a list of registered provider names
func (s *Server) getProviderNames() []string {
	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}
	return names
}
