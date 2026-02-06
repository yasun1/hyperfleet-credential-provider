package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration
type Config struct {
	// Enabled indicates if tracing is enabled
	Enabled bool

	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Endpoint is the OTLP collector endpoint (e.g., "localhost:4317")
	Endpoint string

	// Insecure indicates if the connection should be insecure
	Insecure bool

	// SamplingRatio is the ratio of traces to sample (0.0 to 1.0)
	SamplingRatio float64
}

// DefaultConfig returns default tracing configuration
func DefaultConfig() Config {
	return Config{
		Enabled:       false,
		ServiceName:   "hyperfleet-cloud-provider",
		ServiceVersion: "dev",
		Endpoint:      "localhost:4317",
		Insecure:      true,
		SamplingRatio: 1.0,
	}
}

// Provider wraps the OpenTelemetry tracer provider
type Provider struct {
	tp     *sdktrace.TracerProvider
	tracer trace.Tracer
	config Config
}

// NewProvider creates a new tracing provider
func NewProvider(ctx context.Context, config Config) (*Provider, error) {
	if !config.Enabled {
		// Return a no-op provider
		return &Provider{
			tp:     sdktrace.NewTracerProvider(),
			tracer: otel.Tracer(config.ServiceName),
			config: config,
		}, nil
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP exporter
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(config.Endpoint),
	}
	if config.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create sampler based on sampling ratio
	var sampler sdktrace.Sampler
	if config.SamplingRatio >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SamplingRatio <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SamplingRatio)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		tp:     tp,
		tracer: tp.Tracer(config.ServiceName),
		config: config,
	}, nil
}

// Shutdown shuts down the tracer provider
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tp != nil {
		return p.tp.Shutdown(ctx)
	}
	return nil
}

// Tracer returns the tracer
func (p *Provider) Tracer() trace.Tracer {
	return p.tracer
}

// StartSpan starts a new span with the given name
func (p *Provider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return p.tracer.Start(ctx, name, opts...)
}

// RecordError records an error on the span in the context
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err)
	}
}

// SetAttributes sets attributes on the span in the context
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddEvent adds an event to the span in the context
func AddEvent(ctx context.Context, name string, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, opts...)
	}
}
