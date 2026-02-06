package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the cloud provider
type Metrics struct {
	// Token generation metrics
	TokenRequestsTotal       *prometheus.CounterVec
	TokenGenerationDuration  *prometheus.HistogramVec
	TokenGenerationErrors    *prometheus.CounterVec

	// Credential validation metrics
	CredentialValidationErrors *prometheus.CounterVec

	// Health check metrics
	HealthCheckDuration *prometheus.HistogramVec
	HealthCheckErrors   *prometheus.CounterVec
}

// Config holds configuration for metrics
type Config struct {
	// Namespace for metrics (default: "hyperfleet_cloud_provider")
	Namespace string

	// Subsystem for metrics (default: "")
	Subsystem string

	// Registry to use (default: prometheus.DefaultRegisterer)
	Registry prometheus.Registerer
}

// DefaultConfig returns default metrics configuration
func DefaultConfig() Config {
	return Config{
		Namespace: "hyperfleet_cloud_provider",
		Subsystem: "",
		Registry:  prometheus.DefaultRegisterer,
	}
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(config Config) *Metrics {
	if config.Namespace == "" {
		config.Namespace = "hyperfleet_cloud_provider"
	}
	if config.Registry == nil {
		config.Registry = prometheus.DefaultRegisterer
	}

	factory := promauto.With(config.Registry)

	return &Metrics{
		TokenRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "token_requests_total",
				Help:      "Total number of token generation requests",
			},
			[]string{"provider", "status"},
		),

		TokenGenerationDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "token_generation_duration_seconds",
				Help:      "Token generation duration in seconds",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"provider"},
		),

		TokenGenerationErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "token_generation_errors_total",
				Help:      "Total number of token generation errors",
			},
			[]string{"provider", "error_type"},
		),

		CredentialValidationErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "credential_validation_errors_total",
				Help:      "Total number of credential validation errors",
			},
			[]string{"provider"},
		),

		HealthCheckDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "health_check_duration_seconds",
				Help:      "Health check duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"check_name"},
		),

		HealthCheckErrors: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: config.Namespace,
				Subsystem: config.Subsystem,
				Name:      "health_check_errors_total",
				Help:      "Total number of health check errors",
			},
			[]string{"check_name"},
		),
	}
}

// RecordTokenRequest records a token generation request
func (m *Metrics) RecordTokenRequest(provider, status string) {
	m.TokenRequestsTotal.WithLabelValues(provider, status).Inc()
}

// RecordTokenGenerationDuration records the duration of token generation
func (m *Metrics) RecordTokenGenerationDuration(provider string, duration time.Duration) {
	m.TokenGenerationDuration.WithLabelValues(provider).Observe(duration.Seconds())
}

// RecordTokenGenerationError records a token generation error
func (m *Metrics) RecordTokenGenerationError(provider, errorType string) {
	m.TokenGenerationErrors.WithLabelValues(provider, errorType).Inc()
}

// RecordCredentialValidationError records a credential validation error
func (m *Metrics) RecordCredentialValidationError(provider string) {
	m.CredentialValidationErrors.WithLabelValues(provider).Inc()
}

// RecordHealthCheckDuration records the duration of a health check
func (m *Metrics) RecordHealthCheckDuration(checkName string, duration time.Duration) {
	m.HealthCheckDuration.WithLabelValues(checkName).Observe(duration.Seconds())
}

// RecordHealthCheckError records a health check error
func (m *Metrics) RecordHealthCheckError(checkName string) {
	m.HealthCheckErrors.WithLabelValues(checkName).Inc()
}

// Timer is a helper for timing operations
type Timer struct {
	start time.Time
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// ObserveDuration returns the duration since the timer was created
func (t *Timer) ObserveDuration() time.Duration {
	return time.Since(t.start)
}
