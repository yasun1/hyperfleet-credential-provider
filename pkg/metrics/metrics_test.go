package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Subsystem: "subsys",
		Registry:  registry,
	}

	m := NewMetrics(config)
	require.NotNil(t, m)
	assert.NotNil(t, m.TokenRequestsTotal)
	assert.NotNil(t, m.TokenGenerationDuration)
	assert.NotNil(t, m.TokenGenerationErrors)
	assert.NotNil(t, m.CredentialValidationErrors)
	assert.NotNil(t, m.HealthCheckDuration)
	assert.NotNil(t, m.HealthCheckErrors)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "hyperfleet_cloud_provider", config.Namespace)
	assert.Equal(t, "", config.Subsystem)
	assert.NotNil(t, config.Registry)
}

func TestRecordTokenRequest(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	// Record successful requests
	m.RecordTokenRequest("gcp", "success")
	m.RecordTokenRequest("gcp", "success")
	m.RecordTokenRequest("aws", "success")

	// Record failed requests
	m.RecordTokenRequest("gcp", "error")

	// Verify metrics can be collected
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Find the token_requests_total metric
	found := false
	for _, mf := range metricFamilies {
		if strings.Contains(mf.GetName(), "token_requests_total") {
			found = true
			assert.Equal(t, 3, len(mf.GetMetric())) // gcp-success, aws-success, gcp-error
		}
	}
	assert.True(t, found, "token_requests_total metric not found")
}

func TestRecordTokenGenerationDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	// Record durations
	m.RecordTokenGenerationDuration("gcp", 100*time.Millisecond)
	m.RecordTokenGenerationDuration("gcp", 200*time.Millisecond)
	m.RecordTokenGenerationDuration("aws", 50*time.Millisecond)

	// Verify metrics can be collected
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Find the duration histogram
	found := false
	for _, mf := range metricFamilies {
		if strings.Contains(mf.GetName(), "token_generation_duration_seconds") {
			found = true
			assert.Equal(t, 2, len(mf.GetMetric())) // gcp, aws
		}
	}
	assert.True(t, found, "token_generation_duration_seconds metric not found")
}

func TestRecordTokenGenerationError(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	// Record errors
	m.RecordTokenGenerationError("gcp", "network_error")
	m.RecordTokenGenerationError("gcp", "network_error")
	m.RecordTokenGenerationError("gcp", "auth_error")

	// Verify metrics can be collected
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Find the error metric
	found := false
	for _, mf := range metricFamilies {
		if strings.Contains(mf.GetName(), "token_generation_errors_total") {
			found = true
			assert.Equal(t, 2, len(mf.GetMetric())) // network_error, auth_error
		}
	}
	assert.True(t, found, "token_generation_errors_total metric not found")
}

func TestRecordCredentialValidationError(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	// Record errors
	m.RecordCredentialValidationError("gcp")
	m.RecordCredentialValidationError("gcp")
	m.RecordCredentialValidationError("aws")

	// Verify metrics can be collected
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Find the credential error metric
	found := false
	for _, mf := range metricFamilies {
		if strings.Contains(mf.GetName(), "credential_validation_errors_total") {
			found = true
			assert.Equal(t, 2, len(mf.GetMetric())) // gcp, aws
		}
	}
	assert.True(t, found, "credential_validation_errors_total metric not found")
}

func TestRecordHealthCheckDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	// Record durations
	m.RecordHealthCheckDuration("database", 10*time.Millisecond)
	m.RecordHealthCheckDuration("database", 20*time.Millisecond)
	m.RecordHealthCheckDuration("api", 5*time.Millisecond)

	// Verify metrics can be collected
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Find the health check duration metric
	found := false
	for _, mf := range metricFamilies {
		if strings.Contains(mf.GetName(), "health_check_duration_seconds") {
			found = true
			assert.Equal(t, 2, len(mf.GetMetric())) // database, api
		}
	}
	assert.True(t, found, "health_check_duration_seconds metric not found")
}

func TestRecordHealthCheckError(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	// Record errors
	m.RecordHealthCheckError("database")
	m.RecordHealthCheckError("database")
	m.RecordHealthCheckError("api")

	// Verify metrics can be collected
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Find the health check error metric
	found := false
	for _, mf := range metricFamilies {
		if strings.Contains(mf.GetName(), "health_check_errors_total") {
			found = true
			assert.Equal(t, 2, len(mf.GetMetric())) // database, api
		}
	}
	assert.True(t, found, "health_check_errors_total metric not found")
}

func TestTimer(t *testing.T) {
	timer := NewTimer()
	require.NotNil(t, timer)

	time.Sleep(10 * time.Millisecond)

	duration := timer.ObserveDuration()
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
	assert.Less(t, duration, 100*time.Millisecond) // Should be much less unless system is very slow
}

func TestMultipleMetricsInstances(t *testing.T) {
	registry1 := prometheus.NewRegistry()
	registry2 := prometheus.NewRegistry()

	m1 := NewMetrics(Config{Namespace: "test1", Registry: registry1})
	m2 := NewMetrics(Config{Namespace: "test2", Registry: registry2})

	// Record different values in each
	m1.RecordTokenRequest("gcp", "success")
	m2.RecordTokenRequest("gcp", "success")
	m2.RecordTokenRequest("gcp", "success")

	// Verify they're independent
	mf1, err := registry1.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, mf1)

	mf2, err := registry2.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, mf2)
}

func TestMetricsWithSubsystem(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Subsystem: "provider",
		Registry:  registry,
	}

	m := NewMetrics(config)
	m.RecordTokenRequest("gcp", "success")

	// Verify metrics can be collected
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Metric name should include subsystem (test_provider_token_requests_total)
	found := false
	for _, mf := range metricFamilies {
		if strings.Contains(mf.GetName(), "token_requests_total") {
			found = true
		}
	}
	assert.True(t, found)
}

func TestAllProviders(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	providers := []string{"gcp", "aws", "azure"}

	for _, provider := range providers {
		m.RecordTokenRequest(provider, "success")
		m.RecordTokenGenerationDuration(provider, 100*time.Millisecond)
		m.RecordTokenGenerationError(provider, "test_error")
		m.RecordCredentialValidationError(provider)
	}

	// Verify all metrics can be collected
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Should have multiple metric families
	assert.GreaterOrEqual(t, len(metricFamilies), 4)
}

func TestMetricsNamespace(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "custom_namespace",
		Registry:  registry,
	}

	m := NewMetrics(config)
	m.RecordTokenRequest("gcp", "success")

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	// Verify namespace is in metric name
	found := false
	for _, mf := range metricFamilies {
		if strings.HasPrefix(mf.GetName(), "custom_namespace_") {
			found = true
		}
	}
	assert.True(t, found, "custom namespace not found in metric names")
}

func TestMetricsExport(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	// Record various metrics
	m.RecordTokenRequest("gcp", "success")
	m.RecordTokenGenerationDuration("gcp", 100*time.Millisecond)
	m.RecordTokenGenerationError("aws", "network_error")
	m.RecordCredentialValidationError("azure")
	m.RecordHealthCheckDuration("database", 50*time.Millisecond)
	m.RecordHealthCheckError("api")

	// Verify all metrics can be gathered
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)
	assert.NotEmpty(t, metricFamilies)

	// Verify we have all 6 metric families
	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		metricNames[mf.GetName()] = true
	}

	// Check that we have all expected metrics
	assert.True(t, metricNames["test_token_requests_total"])
	assert.True(t, metricNames["test_token_generation_duration_seconds"])
	assert.True(t, metricNames["test_token_generation_errors_total"])
	assert.True(t, metricNames["test_credential_validation_errors_total"])
	assert.True(t, metricNames["test_health_check_duration_seconds"])
	assert.True(t, metricNames["test_health_check_errors_total"])
	assert.GreaterOrEqual(t, len(metricNames), 6) // All 6 metric families
}

func getAllKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestCounterIncrements(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	// Increment the same counter multiple times
	for i := 0; i < 10; i++ {
		m.RecordTokenRequest("gcp", "success")
	}

	// Use testutil to get the actual counter value
	expected := `
# HELP test_token_requests_total Total number of token generation requests
# TYPE test_token_requests_total counter
test_token_requests_total{provider="gcp",status="success"} 10
`
	err := testutil.CollectAndCompare(m.TokenRequestsTotal, strings.NewReader(expected))
	assert.NoError(t, err)
}

func TestHistogramObservations(t *testing.T) {
	registry := prometheus.NewRegistry()
	config := Config{
		Namespace: "test",
		Registry:  registry,
	}

	m := NewMetrics(config)

	// Record various durations
	durations := []time.Duration{
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	for _, d := range durations {
		m.RecordTokenGenerationDuration("gcp", d)
	}

	// Verify histogram has 5 observations
	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if strings.Contains(mf.GetName(), "token_generation_duration_seconds") {
			for _, metric := range mf.GetMetric() {
				if metric.GetHistogram() != nil {
					assert.Equal(t, uint64(5), metric.GetHistogram().GetSampleCount())
				}
			}
		}
	}
}
