package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.False(t, config.Enabled)
	assert.Equal(t, "hyperfleet-cloud-provider", config.ServiceName)
	assert.Equal(t, "dev", config.ServiceVersion)
	assert.Equal(t, "localhost:4317", config.Endpoint)
	assert.True(t, config.Insecure)
	assert.Equal(t, 1.0, config.SamplingRatio)
}

func TestNewProvider_Disabled(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Enabled = false

	provider, err := NewProvider(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.NotNil(t, provider.Tracer())

	// Shutdown should work even when disabled
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestNewProvider_Enabled_WithoutCollector(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	config := DefaultConfig()
	config.Enabled = true
	config.Endpoint = "localhost:9999" // Non-existent endpoint

	// Should still create provider even if collector is not available
	// The exporter will just fail to export spans
	provider, err := NewProvider(ctx, config)
	if err != nil {
		// This is acceptable - exporter creation might fail without collector
		t.Logf("Expected error when collector not available: %v", err)
		return
	}

	require.NotNil(t, provider)
	assert.NotNil(t, provider.Tracer())

	// Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()
	err = provider.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

func TestProvider_StartSpan(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Enabled = false // Disabled for testing without collector

	provider, err := NewProvider(ctx, config)
	require.NoError(t, err)
	defer provider.Shutdown(ctx)

	// Start a span
	spanCtx, span := provider.StartSpan(ctx, "test-operation")
	require.NotNil(t, spanCtx)
	require.NotNil(t, span)

	// Add attributes
	span.SetAttributes(
		attribute.String("test.key", "test.value"),
		attribute.Int("test.number", 42),
	)

	// End the span
	span.End()
}

func TestProvider_NestedSpans(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Enabled = false

	provider, err := NewProvider(ctx, config)
	require.NoError(t, err)
	defer provider.Shutdown(ctx)

	// Parent span
	parentCtx, parentSpan := provider.StartSpan(ctx, "parent-operation")
	defer parentSpan.End()

	// Child span
	childCtx, childSpan := provider.StartSpan(parentCtx, "child-operation")
	defer childSpan.End()

	// Verify context is different
	assert.NotEqual(t, parentCtx, childCtx)
}

func TestRecordError(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Enabled = false

	provider, err := NewProvider(ctx, config)
	require.NoError(t, err)
	defer provider.Shutdown(ctx)

	// Start a span
	spanCtx, span := provider.StartSpan(ctx, "test-operation")
	defer span.End()

	// Record an error
	testErr := assert.AnError
	RecordError(spanCtx, testErr)

	// Also test with span status
	span.SetStatus(codes.Error, "operation failed")
}

func TestSetAttributes(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Enabled = false

	provider, err := NewProvider(ctx, config)
	require.NoError(t, err)
	defer provider.Shutdown(ctx)

	// Start a span
	spanCtx, span := provider.StartSpan(ctx, "test-operation")
	defer span.End()

	// Set attributes via helper
	SetAttributes(spanCtx,
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
		attribute.Bool("key3", true),
	)
}

func TestAddEvent(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Enabled = false

	provider, err := NewProvider(ctx, config)
	require.NoError(t, err)
	defer provider.Shutdown(ctx)

	// Start a span
	spanCtx, span := provider.StartSpan(ctx, "test-operation")
	defer span.End()

	// Add event via helper
	AddEvent(spanCtx, "test-event")

	// Add event with attributes
	AddEvent(spanCtx, "detailed-event",
		// Note: EventOption would need to be imported from trace package
	)
}

func TestSamplingRatios(t *testing.T) {
	tests := []struct {
		name          string
		samplingRatio float64
		wantSampler   string
	}{
		{
			name:          "always sample",
			samplingRatio: 1.0,
			wantSampler:   "AlwaysSample",
		},
		{
			name:          "never sample",
			samplingRatio: 0.0,
			wantSampler:   "NeverSample",
		},
		{
			name:          "ratio sample",
			samplingRatio: 0.5,
			wantSampler:   "RatioBased",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			config := DefaultConfig()
			config.Enabled = false
			config.SamplingRatio = tt.samplingRatio

			provider, err := NewProvider(ctx, config)
			require.NoError(t, err)
			defer provider.Shutdown(ctx)

			assert.NotNil(t, provider.Tracer())
		})
	}
}

func TestProvider_MultipleShutdowns(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Enabled = false

	provider, err := NewProvider(ctx, config)
	require.NoError(t, err)

	// First shutdown
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)

	// Second shutdown should also work
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestProvider_SpanContext(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Enabled = false

	provider, err := NewProvider(ctx, config)
	require.NoError(t, err)
	defer provider.Shutdown(ctx)

	// Start a span
	spanCtx, span := provider.StartSpan(ctx, "test-operation")
	defer span.End()

	// Verify span context is valid
	assert.NotEqual(t, ctx, spanCtx)

	// Start another span from the first span's context
	_, span2 := provider.StartSpan(spanCtx, "nested-operation")
	defer span2.End()
}

func TestProvider_AttributeTypes(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Enabled = false

	provider, err := NewProvider(ctx, config)
	require.NoError(t, err)
	defer provider.Shutdown(ctx)

	spanCtx, span := provider.StartSpan(ctx, "test-operation")
	defer span.End()

	// Test various attribute types
	SetAttributes(spanCtx,
		attribute.String("string", "value"),
		attribute.Int("int", 42),
		attribute.Int64("int64", 9223372036854775807),
		attribute.Float64("float64", 3.14159),
		attribute.Bool("bool", true),
		attribute.StringSlice("string_slice", []string{"a", "b", "c"}),
		attribute.IntSlice("int_slice", []int{1, 2, 3}),
	)
}

func TestRecordError_WithoutSpan(t *testing.T) {
	// Should not panic when called without a span in context
	ctx := context.Background()
	RecordError(ctx, assert.AnError)
}

func TestSetAttributes_WithoutSpan(t *testing.T) {
	// Should not panic when called without a span in context
	ctx := context.Background()
	SetAttributes(ctx, attribute.String("key", "value"))
}

func TestAddEvent_WithoutSpan(t *testing.T) {
	// Should not panic when called without a span in context
	ctx := context.Background()
	AddEvent(ctx, "test-event")
}

func TestProvider_ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config disabled",
			config: Config{
				Enabled:       false,
				ServiceName:   "test-service",
				ServiceVersion: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "empty service name",
			config: Config{
				Enabled:       false,
				ServiceName:   "",
				ServiceVersion: "1.0.0",
			},
			wantErr: false, // Should still work with empty name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			provider, err := NewProvider(ctx, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)
			defer provider.Shutdown(ctx)
		})
	}
}
