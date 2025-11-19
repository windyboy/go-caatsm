package telemetry

import (
	"caatsm/internal/infra/buildinfo"
	"caatsm/internal/infra/config"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.uber.org/zap"
)

var (
	// instanceID stores a unique identifier for this running process instance.
	// It is initialized once at startup and remains stable for the lifetime of the process.
	instanceID     string
	instanceIDOnce sync.Once
)

// InitOTEL initializes the OpenTelemetry SDK with proper resource attributes,
// sampling configuration, and exporters. This should be called once at application startup.
// logger is optional; if provided, sensitive telemetry configuration will be logged at debug level.
func InitOTEL(ctx context.Context, cfg *config.Config, logger *zap.Logger) error {
	if !cfg.Telemetry.Enabled {
		return nil
	}

	// Log sensitive telemetry configuration to internal debug logs only
	if logger != nil {
		normalizedEndpoint := normalizeEndpoint(cfg.Telemetry.Endpoint)
		logger.Info("Initializing OpenTelemetry",
			zap.String("telemetry.endpoint.original", cfg.Telemetry.Endpoint),
			zap.String("telemetry.endpoint.normalized", normalizedEndpoint),
			zap.Bool("telemetry.insecure", cfg.Telemetry.Insecure),
		)
	}

	// Create resource with comprehensive service information
	res, err := createResource(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create OTEL resource: %w", err)
	}

	// Initialize tracing
	if err := initTracing(ctx, cfg, res); err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Initialize metrics
	if err := initMetrics(ctx, cfg, res); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	return nil
}

// getInstanceID returns a unique identifier for this running process instance.
// It checks environment variables (POD_NAME, CONTAINER_ID, HOSTNAME) first, then generates
// a UUID if no environment variable is available. The ID is initialized once and remains
// stable for the lifetime of the process.
func getInstanceID() string {
	instanceIDOnce.Do(func() {
		// Check for Kubernetes pod name first (most common in containerized deployments)
		if podName := os.Getenv("POD_NAME"); podName != "" {
			instanceID = podName
			return
		}

		// Check for container ID (Docker, containerd, etc.)
		if containerID := os.Getenv("CONTAINER_ID"); containerID != "" {
			instanceID = containerID
			return
		}

		// Check for HOSTNAME (often set in containers)
		if hostname := os.Getenv("HOSTNAME"); hostname != "" {
			// Use hostname if it's not a generic default
			if hostname != "localhost" && hostname != "localhost.localdomain" {
				instanceID = hostname
				return
			}
		}

		// Generate a UUID for this process instance
		instanceID = uuid.NewString()
	})

	return instanceID
}

// createResource creates a resource with standard and custom attributes.
// Sensitive infrastructure details (endpoint, insecure flag) are excluded from resource
// attributes to prevent leakage. These values are logged internally at debug level if
// a logger is provided to InitOTEL.
func createResource(ctx context.Context, cfg *config.Config) (*resource.Resource, error) {
	// Get the runtime instance ID (falls back to buildinfo.Commit if needed)
	runtimeInstanceID := getInstanceID()
	if runtimeInstanceID == "" {
		// Final fallback to build commit if somehow instance ID is empty
		runtimeInstanceID = buildinfo.Commit
	}

	attrs := []attribute.KeyValue{
		// Standard semantic conventions
		semconv.ServiceName("caatsm"),
		semconv.ServiceVersion(buildinfo.Version),
		semconv.ServiceInstanceID(runtimeInstanceID),
		semconv.ServiceNamespace("airport"),

		// Custom attributes
		attribute.String("service.component", "receiver"),
		attribute.String("service.environment", getEnvironment()),
		attribute.String("build.commit", buildinfo.Commit),
		attribute.String("build.built_at", buildinfo.BuiltAt),
	}

	// Add non-sensitive indicator for telemetry endpoint configuration
	// (without exposing the actual endpoint value)
	if cfg.Telemetry.Endpoint != "" {
		attrs = append(attrs, attribute.Bool("telemetry.endpoint.configured", true))
	} else {
		attrs = append(attrs, attribute.Bool("telemetry.endpoint.configured", false))
	}

	return resource.New(ctx, resource.WithAttributes(attrs...))
}

// normalizeEndpoint removes the scheme from the endpoint, returning just host:port
// The OpenTelemetry SDK's WithEndpoint() expects host:port, and WithInsecure() controls the protocol
func normalizeEndpoint(endpoint string) string {
	if endpoint == "" {
		return endpoint
	}

	endpoint = strings.TrimSpace(endpoint)
	
	// Remove http:// or https:// scheme if present
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	
	return endpoint
}

// initTracing sets up the trace provider with appropriate sampling
func initTracing(ctx context.Context, cfg *config.Config, res *resource.Resource) error {
	var traceExporterOptions []otlptracehttp.Option

	// Normalize endpoint to remove scheme (WithEndpoint expects host:port)
	endpoint := normalizeEndpoint(cfg.Telemetry.Endpoint)
	traceExporterOptions = append(traceExporterOptions, otlptracehttp.WithEndpoint(endpoint))

	if cfg.Telemetry.Insecure {
		// Use HTTP instead of HTTPS when insecure is true
		traceExporterOptions = append(traceExporterOptions, otlptracehttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceExporterOptions...)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Configure sampling based on environment
	sampler := getSamplerForEnvironment(getEnvironment())

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(1*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithMaxQueueSize(2048),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
	)

	otel.SetTracerProvider(tracerProvider)
	return nil
}

// initMetrics sets up the meter provider
func initMetrics(ctx context.Context, cfg *config.Config, res *resource.Resource) error {
	var metricExporterOptions []otlpmetrichttp.Option

	// Normalize endpoint to remove scheme (WithEndpoint expects host:port)
	endpoint := normalizeEndpoint(cfg.Telemetry.Endpoint)
	metricExporterOptions = append(metricExporterOptions, otlpmetrichttp.WithEndpoint(endpoint))

	if cfg.Telemetry.Insecure {
		// Use HTTP instead of HTTPS when insecure is true
		metricExporterOptions = append(metricExporterOptions, otlpmetrichttp.WithInsecure())
	}

	metricExporter, err := otlpmetrichttp.New(ctx, metricExporterOptions...)
	if err != nil {
		return fmt.Errorf("failed to create metric exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
			sdkmetric.WithInterval(30*time.Second),
		)),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(meterProvider)
	return nil
}

// getSamplerForEnvironment returns appropriate sampling strategy for each environment
func getSamplerForEnvironment(env string) sdktrace.Sampler {
	switch env {
	case "prod", "production":
		// 1% sampling in production to control costs and performance
		return sdktrace.TraceIDRatioBased(0.01)
	case "staging":
		// 10% sampling in staging for better observability
		return sdktrace.TraceIDRatioBased(0.1)
	case "test", "testing":
		// Always sample in testing for complete coverage
		return sdktrace.AlwaysSample()
	default:
		// 100% sampling in development for debugging
		return sdktrace.AlwaysSample()
	}
}

// ShutdownOTEL gracefully shuts down the OTEL providers
func ShutdownOTEL(ctx context.Context) error {
	var errs []error

	if tracerProvider, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown tracer provider: %w", err))
		}
	}

	if meterProvider, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider); ok {
		if err := meterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown meter provider: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("OTEL shutdown errors: %v", errs)
	}
	return nil
}

// getEnvironment returns the current environment from GO_ENV
func getEnvironment() string {
	if env := os.Getenv("GO_ENV"); env != "" {
		return env
	}
	return "dev"
}
