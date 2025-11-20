package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nauticalab/devenv-engine/internal/auth"
	"github.com/nauticalab/devenv-engine/internal/k8s"
)

// Server represents the HTTP API server
type Server struct {
	router    *chi.Mux
	handler   *Handler
	providers map[string]auth.AuthProvider
	addr      string
}

// ServerConfig holds configuration for the API server
type ServerConfig struct {
	Port      int
	Audience  string
	K8sClient *k8s.Client
	Version   string
	GitCommit string
	BuildTime string
	GoVersion string
}

// NewServer creates a new API server with the given configuration
func NewServer(config ServerConfig) (*Server, error) {
	// Create K8s SA auth provider
	k8sProvider := auth.NewK8sSAProvider(config.K8sClient, config.Audience, "devenv-{developer}")

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
		router:    router,
		handler:   handler,
		providers: providers,
		addr:      addr,
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

	// Auth middleware (for protected routes)
	authMiddleware := auth.Middleware(providers)

	// Public routes (no auth)
	router.Group(func(r chi.Router) {
		r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
			respondSuccess(w, HealthResponse{
				Status:    "ok",
				Timestamp: time.Now(),
			})
		})
	})

	// Protected routes (require auth)
	router.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		// Routes will be added in setupRoutes
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
