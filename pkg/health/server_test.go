package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

func TestNewServer(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)
	require.NotNil(t, server)
	assert.Equal(t, config.Address, server.addr)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.checks)
}

func TestServerStartStop(t *testing.T) {
	config := DefaultConfig()
	config.Address = ":18080" // Use different port to avoid conflicts
	config.Logger = logger.Nop()

	server := NewServer(config)

	// Start server
	err := server.Start()
	require.NoError(t, err)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	assert.NoError(t, err)
}

func TestHandleLiveness(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	server.handleLiveness(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response HealthResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
	assert.Contains(t, response.Checks, "server")
	assert.Equal(t, "running", response.Checks["server"])
}

func TestHandleReadiness_NoChecks(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	server.handleReadiness(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response HealthResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
}

func TestHandleReadiness_HealthyChecks(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	// Register healthy checks
	server.RegisterCheck("check1", AlwaysHealthy())
	server.RegisterCheck("check2", AlwaysHealthy())

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	server.handleReadiness(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response HealthResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "ok", response.Checks["check1"])
	assert.Equal(t, "ok", response.Checks["check2"])
}

func TestHandleReadiness_UnhealthyChecks(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	// Register mixed checks
	server.RegisterCheck("healthy", AlwaysHealthy())
	server.RegisterCheck("unhealthy", AlwaysUnhealthy("test failure"))

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	server.handleReadiness(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response HealthResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "degraded", response.Status)
	assert.Equal(t, "ok", response.Checks["healthy"])
	assert.Contains(t, response.Checks["unhealthy"], "failed")
	assert.Contains(t, response.Checks["unhealthy"], "test failure")
}

func TestHandleRoot(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.handleRoot(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "hyperfleet-cloud-provider", response["service"])
	assert.Equal(t, "running", response["status"])
	assert.NotNil(t, response["uptime"])
	assert.NotNil(t, response["endpoints"])
}

func TestHandleRoot_NotFound(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w := httptest.NewRecorder()

	server.handleRoot(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRegisterCheck(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	// Register a check
	check := AlwaysHealthy()
	server.RegisterCheck("test", check)

	// Verify it was registered
	server.mu.RLock()
	_, exists := server.checks["test"]
	server.mu.RUnlock()

	assert.True(t, exists)
}

func TestRegisterCheck_Multiple(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	// Register multiple checks
	server.RegisterCheck("check1", AlwaysHealthy())
	server.RegisterCheck("check2", AlwaysHealthy())
	server.RegisterCheck("check3", AlwaysHealthy())

	// Verify all were registered
	server.mu.RLock()
	count := len(server.checks)
	server.mu.RUnlock()

	assert.Equal(t, 3, count)
}

func TestHealthResponse_JSON(t *testing.T) {
	response := HealthResponse{
		Status: "ok",
		Checks: map[string]string{
			"check1": "ok",
			"check2": "ok",
		},
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var decoded HealthResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, response.Status, decoded.Status)
	assert.Equal(t, response.Checks, decoded.Checks)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, ":8080", config.Address)
	assert.Equal(t, 5*time.Second, config.ReadTimeout)
	assert.Equal(t, 10*time.Second, config.WriteTimeout)
}

func TestServer_ConcurrentChecks(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	// Register slow checks
	slowCheck := func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	for i := 0; i < 10; i++ {
		server.RegisterCheck(fmt.Sprintf("check%d", i), slowCheck)
	}

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	server.handleReadiness(w, req)
	duration := time.Since(start)

	// All checks should run serially, so total time should be ~100ms
	assert.Less(t, duration, 200*time.Millisecond)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestServer_CheckTimeout(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	// Register a check that respects context cancellation
	timeoutCheck := func(ctx context.Context) error {
		select {
		case <-time.After(10 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	server.RegisterCheck("timeout", timeoutCheck)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	server.handleReadiness(w, req)
	duration := time.Since(start)

	// Should timeout in ~5 seconds (server timeout)
	assert.Less(t, duration, 6*time.Second)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestServer_LivezAlias(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	// Test /livez endpoint (alias for /healthz)
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	w := httptest.NewRecorder()

	server.handleLiveness(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response HealthResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response.Status)
}

func TestMetricsEndpoint(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

func TestRootEndpointIncludesMetrics(t *testing.T) {
	config := DefaultConfig()
	config.Logger = logger.Nop()

	server := NewServer(config)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.handleRoot(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	endpoints, ok := response["endpoints"].([]interface{})
	require.True(t, ok)

	// Convert to strings
	endpointStrs := make([]string, len(endpoints))
	for i, ep := range endpoints {
		endpointStrs[i] = ep.(string)
	}

	assert.Contains(t, endpointStrs, "/metrics")
}
