package main

import (
	"caatsm/internal/infra/buildinfo"
	"caatsm/internal/infra/config"
	"caatsm/pkg/di"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// main is the entrypoint for the caatsm CLI.
// It delegates all logic to run so that startup behaviour can be tested.
func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "caatsm failed: %v\n", err)
		os.Exit(1)
	}
}

// run constructs the CLI application and executes it with the provided args.
func run(args []string) error {
	app := setupApp()
	return app.Run(args)
}

func setupApp() *cli.App {
	return &cli.App{
		Name:  "telegram message process",
		Usage: "A Civil Aviation Authority Telegram Message Processor",
		Commands: []*cli.Command{
			{
				Name:  "listen",
				Usage: "Listen to nats messages",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "nats-url",
						Aliases: []string{"n"},
						Usage:   "NATS server address",
						EnvVars: []string{"NATS_SERVER"},
					},
					&cli.StringFlag{
						Name:    "subject",
						Aliases: []string{"t"},
						Usage:   "NATS subject to listen to",
						EnvVars: []string{"NATS_SUBJECT"},
					},
					&cli.StringFlag{
						Name:  "stream",
						Usage: "NATS JetStream stream name",
					},
					&cli.StringFlag{
						Name:  "nats-mode",
						Usage: "NATS mode: jetstream or core",
					},
					&cli.StringFlag{
						Name:  "consumer",
						Usage: "NATS JetStream durable consumer",
					},
					&cli.StringFlag{
						Name:  "publisher-topic",
						Usage: "Subject used by the publisher",
					},
					&cli.StringFlag{
						Name:  "postgres-url",
						Usage: "PostgreSQL connection URL",
						EnvVars: []string{
							"POSTGRES_URL",
						},
					},
					&cli.StringFlag{
						Name:  "log-level",
						Usage: "Logger level (debug, info, warn, error)",
					},
					&cli.StringFlag{
						Name:  "replay-from",
						Usage: "Override deliver policy: all|new|last|seq:<n>|time:<RFC3339>",
					},
					&cli.DurationFlag{
						Name:  "ack-wait",
						Usage: "Override consumer ack wait duration",
					},
					&cli.BoolFlag{
						Name:  "telemetry-enabled",
						Usage: "Enable OpenTelemetry exporters",
					},
					&cli.StringFlag{
						Name:  "telemetry-endpoint",
						Usage: "OpenTelemetry collector OTLP endpoint",
					},
					&cli.BoolFlag{
						Name:  "telemetry-insecure",
						Usage: "Send OTLP data without TLS",
					},
					&cli.StringFlag{
						Name:    "monitoring-addr",
						Usage:   "Monitoring server address (e.g., 192.168.1.100:2112 or :2112)",
						EnvVars: []string{"CAATSM_MONITORING_ADDR"},
					},
					&cli.BoolFlag{
						Name:    "monitoring-disabled",
						Usage:   "Disable monitoring server",
						EnvVars: []string{"CAATSM_MONITORING_DISABLED"},
					},
				},
				Action: executeListen,
			},
		},
	}
}

// executeListen is the CLI handler for the "listen" command.
// It is responsible for loading configuration, applying CLI overrides,
// and delegating the main processing lifecycle to runListen.
func executeListen(c *cli.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	applyCLIOverrides(cfg, c)

	// Re-validate configuration after applying CLI overrides to ensure
	// the resulting configuration is still consistent.
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration after CLI overrides: %w", err)
	}

	return runListen(context.Background(), cfg)
}

// runListen coordinates telemetry initialisation, dependency wiring,
// signal handling and graceful shutdown for the listener workflow.
func runListen(parentCtx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config must not be nil")
	}

	ctx, stop := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var shutdownTelemetry func(context.Context) error
	if cfg.Telemetry.Enabled {
		var telErr error
		shutdownTelemetry, telErr = initTelemetry(ctx, cfg)
		if telErr != nil {
			return fmt.Errorf("failed to initialize telemetry: %w", telErr)
		}
		defer func() {
			if err := shutdownTelemetry(context.Background()); err != nil {
				zap.L().Error("Failed to shutdown telemetry", zap.Error(err))
			}
		}()
	}

	// Initialize dependencies using Wire
	processor, consumer, monitorServer, err := di.InitializeAppWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize app: %w", err)
	}

	if monitorServer != nil {
		if err := monitorServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start monitoring server: %w", err)
		}
		defer func() {
			if err := monitorServer.Shutdown(context.Background()); err != nil {
				zap.L().Error("Failed to shutdown monitoring server", zap.Error(err))
			}
		}()
	}

	// Start consumer in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errChan <- fmt.Errorf("consumer error: %w", err)
		}
	}()

	var runErr error

	// Wait for shutdown signal or consumer error
	select {
	case <-ctx.Done():
		zap.L().Info("Received shutdown signal, shutting down",
			zap.Error(ctx.Err()))
	case err := <-errChan:
		if err != nil {
			runErr = err
		}
		// Ensure all downstream users of ctx see cancellation.
		stop()
	}

	// After cancellation, give the consumer a chance to finish cleanup.
	// Consumer checks context before fetch and between messages, so it should exit quickly.
	// Worst case: finishing current message (up to 1s for slow DB) + cleanup (~100ms)
	// 1.5s provides safe buffer while keeping shutdown responsive.
	waitTimeout := 1500 * time.Millisecond
	select {
	case err := <-errChan:
		if err != nil && !errors.Is(err, context.Canceled) {
			runErr = err
		}
	case <-time.After(waitTimeout):
		zap.L().Warn("Timed out waiting for consumer shutdown",
			zap.Duration("timeout", waitTimeout))
	}

	// Use a short timeout for shutdown since consumer should already be stopped
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()
	if err := consumer.Shutdown(shutdownCtx); err != nil {
		if runErr == nil {
			runErr = fmt.Errorf("failed to drain NATS connection: %w", err)
		}
	}

	// Note: processor is initialized but not directly used here
	_ = processor

	return runErr
}

func applyCLIOverrides(cfg *config.Config, c *cli.Context) {
	if cfg == nil || c == nil {
		return
	}
	if flagURL := c.String("nats-url"); flagURL != "" {
		cfg.NATS.URL = flagURL
	}
	if flagTopic := c.String("subject"); flagTopic != "" {
		cfg.Subscription.Topic = flagTopic
	}
	if stream := c.String("stream"); stream != "" {
		cfg.NATS.Stream = stream
	}
	if mode := c.String("nats-mode"); mode != "" {
		cfg.NATS.Mode = strings.ToLower(mode)
	}
	if consumer := c.String("consumer"); consumer != "" {
		cfg.NATS.Consumer = consumer
	}
	if publisherTopic := c.String("publisher-topic"); publisherTopic != "" {
		cfg.Publisher.Topic = publisherTopic
	}
	if pgURL := c.String("postgres-url"); pgURL != "" {
		cfg.Postgres.URL = pgURL
	}
	if logLevel := c.String("log-level"); logLevel != "" {
		cfg.Log.Level = logLevel
	}
	if replay := c.String("replay-from"); replay != "" {
		applyReplayOverride(cfg, replay)
	}
	if c.IsSet("ack-wait") {
		if ack := c.Duration("ack-wait"); ack > 0 {
			cfg.Timeouts.AckWait = ack
			cfg.NATS.ConsumerRules.AckWait = ack
		}
	}
	if c.IsSet("telemetry-enabled") {
		cfg.Telemetry.Enabled = c.Bool("telemetry-enabled")
	}
	if endpoint := c.String("telemetry-endpoint"); endpoint != "" {
		cfg.Telemetry.Endpoint = endpoint
	}
	if c.IsSet("telemetry-insecure") {
		cfg.Telemetry.Insecure = c.Bool("telemetry-insecure")
	}
	if addr := c.String("monitoring-addr"); addr != "" {
		cfg.Monitoring.Addr = addr
		cfg.Monitoring.Disabled = false
	}
	if c.IsSet("monitoring-disabled") {
		cfg.Monitoring.Disabled = c.Bool("monitoring-disabled")
	}
}

func applyReplayOverride(cfg *config.Config, value string) {
	if cfg == nil {
		return
	}
	lower := strings.ToLower(strings.TrimSpace(value))
	switch {
	case lower == "new":
		cfg.NATS.ConsumerRules.DeliverPolicy = "new"
		cfg.NATS.ConsumerRules.StartSequence = 0
		cfg.NATS.ConsumerRules.StartTime = ""
	case lower == "all":
		cfg.NATS.ConsumerRules.DeliverPolicy = "all"
		cfg.NATS.ConsumerRules.StartSequence = 0
		cfg.NATS.ConsumerRules.StartTime = ""
	case lower == "last":
		cfg.NATS.ConsumerRules.DeliverPolicy = "last"
		cfg.NATS.ConsumerRules.StartSequence = 0
		cfg.NATS.ConsumerRules.StartTime = ""
	case lower == "last_per_subject":
		cfg.NATS.ConsumerRules.DeliverPolicy = "last_per_subject"
		cfg.NATS.ConsumerRules.StartSequence = 0
		cfg.NATS.ConsumerRules.StartTime = ""
	case strings.HasPrefix(lower, "seq:"):
		seqStr := strings.TrimPrefix(lower, "seq:")
		if seq, err := strconv.ParseUint(seqStr, 10, 64); err == nil {
			cfg.NATS.ConsumerRules.DeliverPolicy = "sequence"
			cfg.NATS.ConsumerRules.StartSequence = seq
			cfg.NATS.ConsumerRules.StartTime = ""
		}
	case strings.HasPrefix(lower, "time:"):
		ts := strings.TrimSpace(value[5:])
		if _, err := time.Parse(time.RFC3339, ts); err == nil {
			cfg.NATS.ConsumerRules.DeliverPolicy = "time"
			cfg.NATS.ConsumerRules.StartSequence = 0
			cfg.NATS.ConsumerRules.StartTime = ts
		}
	}
}

func initTelemetry(ctx context.Context, cfg *config.Config) (func(context.Context) error, error) {
	if cfg == nil || !cfg.Telemetry.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	if cfg.Telemetry.Endpoint == "" {
		return nil, fmt.Errorf("telemetry endpoint is required when telemetry.enabled=true")
	}

	traceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Telemetry.Endpoint),
		otlptracehttp.WithURLPath("/v1/traces"),
	}

	metricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.Telemetry.Endpoint),
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

	// Get environment for sampling configuration
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "dev"
	}

	// Create comprehensive resource with service information
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithContainer(),
		resource.WithAttributes(
			semconv.ServiceName("caatsm"),
			semconv.ServiceVersion(buildinfo.Version),
			semconv.ServiceNamespace("airport"),
			attribute.String("service.component", "receiver"),
			attribute.String("deployment.environment", env),
			attribute.String("telemetry.endpoint", cfg.Telemetry.Endpoint),
			attribute.Bool("telemetry.insecure", cfg.Telemetry.Insecure),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("build telemetry resource: %w", err)
	}

	// Configure sampling based on environment
	sampler := getSamplerForEnvironment(env)

	// Configure tracer provider with batching and sampling
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithBatchTimeout(1*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithMaxQueueSize(2048),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
	)

	// Configure meter provider with periodic reader
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
			sdkmetric.WithInterval(30*time.Second),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	shutdown := func(ctx context.Context) error {
		errs := []error{}
		if err := mp.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}

	return shutdown, nil
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
