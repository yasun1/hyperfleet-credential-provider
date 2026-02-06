package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server provides health check endpoints
type Server struct {
	addr      string
	server    *http.Server
	logger    logger.Logger
	checks    map[string]Check
	mu        sync.RWMutex
	startTime time.Time
}

// Check represents a health check function
type Check func(ctx context.Context) error

// Config holds server configuration
type Config struct {
	// Address to listen on (e.g., ":8080")
	Address string

	// ReadTimeout for HTTP requests
	ReadTimeout time.Duration

	// WriteTimeout for HTTP responses
	WriteTimeout time.Duration

	// Logger for health server
	Logger logger.Logger
}

// DefaultConfig returns default health server configuration
func DefaultConfig() Config {
	return Config{
		Address:      ":8080",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// NewServer creates a new health check server
func NewServer(config Config) *Server {
	if config.Logger == nil {
		config.Logger = logger.Nop()
	}

	s := &Server{
		addr:      config.Address,
		logger:    config.Logger,
		checks:    make(map[string]Check),
		startTime: time.Now(),
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleLiveness)
	mux.HandleFunc("/readyz", s.handleReadiness)
	mux.HandleFunc("/livez", s.handleLiveness) // Alias for /healthz
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", s.handleRoot)

	s.server = &http.Server{
		Addr:         config.Address,
		Handler:      mux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return s
}

// RegisterCheck adds a named health check
func (s *Server) RegisterCheck(name string, check Check) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[name] = check
	s.logger.Info("Registered health check",
		logger.String("name", name),
	)
}

// Start starts the health check server
func (s *Server) Start() error {
	s.logger.Info("Starting health server",
		logger.String("address", s.addr),
	)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Health server error",
				logger.String("error", err.Error()),
			)
		}
	}()

	return nil
}

// Stop gracefully shuts down the health check server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping health server")
	return s.server.Shutdown(ctx)
}

// handleRoot provides basic information
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	uptime := time.Since(s.startTime)

	response := map[string]interface{}{
		"service":   "hyperfleet-cloud-provider",
		"status":    "running",
		"uptime":    uptime.String(),
		"endpoints": []string{"/healthz", "/readyz", "/livez", "/metrics"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleLiveness handles liveness probe requests
// Liveness probe checks if the application is running
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Liveness is simple - if we can respond, we're alive
	response := HealthResponse{
		Status: "ok",
		Checks: map[string]string{
			"server": "running",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleReadiness handles readiness probe requests
// Readiness probe checks if the application is ready to serve traffic
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	s.mu.RLock()
	checks := make(map[string]Check, len(s.checks))
	for name, check := range s.checks {
		checks[name] = check
	}
	s.mu.RUnlock()

	// If no checks registered, we're ready
	if len(checks) == 0 {
		response := HealthResponse{
			Status: "ok",
			Checks: map[string]string{
				"server": "ready",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Run all health checks
	results := make(map[string]string)
	allHealthy := true

	for name, check := range checks {
		if err := check(ctx); err != nil {
			results[name] = fmt.Sprintf("failed: %s", err.Error())
			allHealthy = false
			s.logger.Warn("Health check failed",
				logger.String("check", name),
				logger.String("error", err.Error()),
			)
		} else {
			results[name] = "ok"
		}
	}

	status := "ok"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status: status,
		Checks: results,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}
