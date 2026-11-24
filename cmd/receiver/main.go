package main

import (
	"caatsm/internal/config"
	natsinfra "caatsm/internal/nats"
	"caatsm/internal/observability"
	"caatsm/internal/pipeline"
	"caatsm/internal/postgres"
	"caatsm/internal/telegram"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "caatsm failed: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	return setupApp().Run(args)
}

func setupApp() *cli.App {
	return &cli.App{
		Name:  "caatsm-receiver",
		Usage: "Civil Aviation Authority Telegram Message Processor",
		Commands: []*cli.Command{
			{
				Name:  "listen",
				Usage: "Listen to NATS JetStream messages",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "nats-url", Aliases: []string{"n"}, Usage: "NATS server address", EnvVars: []string{"NATS_SERVER"}},
					&cli.StringFlag{Name: "subject", Aliases: []string{"t"}, Usage: "NATS subject to listen to", EnvVars: []string{"NATS_SUBJECT"}},
					&cli.StringFlag{Name: "stream", Usage: "NATS JetStream stream name"},
					&cli.StringFlag{Name: "nats-mode", Usage: "NATS mode (jetstream)"},
					&cli.StringFlag{Name: "consumer", Usage: "NATS JetStream durable consumer"},
					&cli.StringFlag{Name: "publisher-topic", Usage: "Subject used by the publisher"},
					&cli.StringFlag{Name: "postgres-url", Usage: "PostgreSQL connection URL", EnvVars: []string{"POSTGRES_URL"}},
					&cli.StringFlag{Name: "log-level", Usage: "Logger level (debug, info, warn, error)"},
					&cli.StringFlag{Name: "replay-from", Usage: "Override deliver policy: all|new|last|last_per_subject|seq:<n>|time:<RFC3339>"},
					&cli.DurationFlag{Name: "ack-wait", Usage: "Override consumer ack wait duration"},
					&cli.BoolFlag{Name: "telemetry-enabled", Usage: "Enable OpenTelemetry exporters"},
					&cli.StringFlag{Name: "telemetry-endpoint", Usage: "OpenTelemetry collector OTLP endpoint"},
					&cli.BoolFlag{Name: "telemetry-insecure", Usage: "Send OTLP data without TLS"},
					&cli.StringFlag{Name: "monitoring-addr", Usage: "Monitoring server address (e.g., 0.0.0.0:2112 or :2112)", EnvVars: []string{"CAATSM_MONITORING_ADDR"}},
					&cli.BoolFlag{Name: "monitoring-disabled", Usage: "Disable monitoring server", EnvVars: []string{"CAATSM_MONITORING_DISABLED"}},
				},
				Action: executeListen,
			},
		},
	}
}

func executeListen(c *cli.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	applyCLIOverrides(cfg, c)
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration after CLI overrides: %w", err)
	}

	return runListen(context.Background(), cfg)
}

// runListen explicitly wires all dependencies and manages reverse-order graceful shutdown.
func runListen(parentCtx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config must not be nil")
	}

	ctx, stop := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger, err := observability.NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	shutdownOTEL, err := observability.InitOTEL(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := shutdownOTEL(shutCtx); err != nil {
			logger.Error("Failed to shutdown telemetry", zap.Error(err))
		}
	}()

	pool, err := postgres.OpenPool(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize postgres: %w", err)
	}
	defer pool.Close()

	store := postgres.NewRepository(pool, logger)

	nc, err := natsinfra.Connect(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize NATS connection: %w", err)
	}
	defer nc.Close()

	js, err := natsinfra.JetStream(nc, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	pub, err := natsinfra.NewPublisher(js, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize publisher: %w", err)
	}

	recorder := observability.NewRecorder(cfg)
	router := telegram.NewRouter()
	aftnValidator := telegram.NewAFTNValidator()
	processor := pipeline.NewProcessor(
		router,
		aftnValidator,
		store,
		pub,
		recorder,
		logger,
		pipeline.Options{ValidateAFTN: cfg.AFTN.ValidationEnabled},
	)

	consumer, err := natsinfra.NewConsumer(nc, js, processor, cfg, recorder, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize consumer: %w", err)
	}

	monitorServer, err := observability.NewServer(cfg, logger, pool, nc)
	if err != nil {
		return fmt.Errorf("failed to initialize monitoring server: %w", err)
	}
	if monitorServer != nil {
		if err := monitorServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start monitoring server: %w", err)
		}
		defer func() {
			shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := monitorServer.Shutdown(shutCtx); err != nil {
				logger.Error("Failed to shutdown monitoring server", zap.Error(err))
			}
		}()
	}

	errChan := make(chan error, 1)
	go func() {
		if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errChan <- fmt.Errorf("consumer error: %w", err)
		} else {
			errChan <- nil
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		logger.Info("Received shutdown signal, shutting down", zap.Error(ctx.Err()))
	case err := <-errChan:
		if err != nil {
			runErr = err
		}
		stop()
	}

	waitTimeout := cfg.App.BatchTimeout + time.Second
	if waitTimeout < 3*time.Second {
		waitTimeout = 3 * time.Second
	}
	select {
	case err := <-errChan:
		if err != nil && !errors.Is(err, context.Canceled) && runErr == nil {
			runErr = err
		}
	case <-time.After(waitTimeout):
		logger.Warn("Timed out waiting for consumer loop exit", zap.Duration("timeout", waitTimeout))
	}

	drainTimeout := cfg.Timeouts.Close
	if drainTimeout <= 0 {
		drainTimeout = 3 * time.Second
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), drainTimeout)
	defer shutdownCancel()
	if err := consumer.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = fmt.Errorf("failed to drain NATS connection: %w", err)
	}

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
