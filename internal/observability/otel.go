package observability

import (
	"caatsm/internal/buildinfo"
	"caatsm/internal/config"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

var (
	instanceID     string
	instanceIDOnce sync.Once
)

// InitOTEL initializes the OpenTelemetry trace and metric SDKs and returns a shutdown hook.
func InitOTEL(ctx context.Context, cfg *config.Config, logger *zap.Logger) (func(context.Context) error, error) {
	if cfg == nil || !cfg.Telemetry.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	if cfg.Telemetry.Endpoint == "" {
		return nil, fmt.Errorf("telemetry endpoint is required when telemetry.enabled=true")
	}

	endpoint := normalizeEndpoint(cfg.Telemetry.Endpoint)
	if logger != nil {
		logger.Info("Initializing OpenTelemetry",
			zap.String("telemetry.endpoint", endpoint),
			zap.Bool("telemetry.insecure", cfg.Telemetry.Insecure),
		)
	}

	res, err := createResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("build telemetry resource: %w", err)
	}

	traceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath("/v1/traces"),
	}
	metricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithURLPath("/v1/metrics"),
	}
	if cfg.Telemetry.Insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, fmt.Errorf("init trace exporter: %w", err)
	}
	metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		return nil, fmt.Errorf("init metric exporter: %w", err)
	}

	sampler := getSamplerForEnvironment(getEnvironment())
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(1*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithMaxQueueSize(2048),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
	)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
			sdkmetric.WithInterval(30*time.Second),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	shutdown := func(shutCtx context.Context) error {
		return errors.Join(mp.Shutdown(shutCtx), tp.Shutdown(shutCtx))
	}
	return shutdown, nil
}

func createResource(ctx context.Context, cfg *config.Config) (*resource.Resource, error) {
	env := getEnvironment()
	runtimeInstanceID := getInstanceID()
	if runtimeInstanceID == "" {
		runtimeInstanceID = buildinfo.Commit
	}

	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceName("caatsm"),
			semconv.ServiceVersion(buildinfo.Version),
			semconv.ServiceInstanceID(runtimeInstanceID),
			semconv.ServiceNamespace("airport"),
			attribute.String("service.component", "receiver"),
			attribute.String("deployment.environment", env),
			attribute.String("build.commit", buildinfo.Commit),
			attribute.String("build.built_at", buildinfo.BuiltAt),
			attribute.Bool("telemetry.endpoint.configured", cfg.Telemetry.Endpoint != ""),
		),
	)
}

func getInstanceID() string {
	instanceIDOnce.Do(func() {
		if podName := os.Getenv("POD_NAME"); podName != "" {
			instanceID = podName
			return
		}
		if containerID := os.Getenv("CONTAINER_ID"); containerID != "" {
			instanceID = containerID
			return
		}
		if hostname := os.Getenv("HOSTNAME"); hostname != "" && hostname != "localhost" && hostname != "localhost.localdomain" {
			instanceID = hostname
			return
		}
		var b [8]byte
		_, _ = rand.Read(b[:])
		instanceID = fmt.Sprintf("%x", b)
	})
	return instanceID
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return strings.TrimSuffix(endpoint, "/")
}

func getSamplerForEnvironment(env string) sdktrace.Sampler {
	switch strings.ToLower(env) {
	case "prod", "production":
		return sdktrace.TraceIDRatioBased(0.01)
	case "staging":
		return sdktrace.TraceIDRatioBased(0.1)
	default:
		return sdktrace.AlwaysSample()
	}
}

func getEnvironment() string {
	if env := os.Getenv("GO_ENV"); env != "" {
		return env
	}
	return "dev"
}
