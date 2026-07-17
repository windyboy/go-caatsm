//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"caatsm/internal/adapter/dto"
	"caatsm/internal/adapter/parser"
	weatherparser "caatsm/internal/adapter/parser/weather"
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	loginfra "caatsm/internal/infra/log"
	natsinfra "caatsm/internal/infra/nats"
	postgresinfra "caatsm/internal/infra/postgres"
	telemetryinfra "caatsm/internal/infra/telemetry"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestJetStreamToTimescaleFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pgContainer, pgURL := startPostgres(ctx, t)
	defer func() {
		_ = pgContainer.Terminate(context.Background())
	}()

	natsContainer, natsURL := startNATS(ctx, t)
	defer func() {
		_ = natsContainer.Terminate(context.Background())
	}()

	cfg := buildTestConfig(natsURL, pgURL)
	logger, err := loginfra.ProvideLogger(cfg)
	if err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	pool, err := postgresinfra.ProvideDB(cfg, logger)
	if err != nil {
		t.Fatalf("failed to init postgres: %v", err)
	}
	defer pool.Close()

	if err := applyDDL(ctx, pool); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	repo, err := postgresinfra.ProvideRepository(pool, logger)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	conn, err := natsinfra.ProvideNATSConn(cfg, logger)
	if err != nil {
		t.Fatalf("failed to connect nats: %v", err)
	}
	defer conn.Drain()

	js, err := natsinfra.ProvideJetStream(conn, cfg, logger)
	if err != nil {
		t.Fatalf("failed to init jetstream: %v", err)
	}

	publisher, err := natsinfra.ProvidePublisher(js, conn, cfg, logger)
	if err != nil {
		t.Fatalf("failed to init publisher: %v", err)
	}

	telemetryRecorder := telemetryinfra.NewNoop()
	proc := app.NewMessageProcessor(parser.ProvideParser(weatherparser.NewWeatherParser()), repo, publisher, telemetryRecorder, logger, cfg)
	consumer, err := natsinfra.ProvideConsumer(conn, js, proc, cfg, telemetryRecorder, logger)
	if err != nil {
		t.Fatalf("failed to init consumer: %v", err)
	}

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumer.Start(runCtx)
	}()
	defer func() {
		runCancel()
		select {
		case <-errCh:
		case <-time.After(2 * time.Second):
		}
		_ = consumer.Shutdown(context.Background())
	}()

	// Publish a message to the input subject.
	payload := []byte(`ZCZC TMQ2526 141605
FF ZBTJZPZX
141604 ZBACZQZX
(ARR-JAE7433/A0132-RKSI-ZBTJ1604)
NNNN`)

	msg := nats.NewMsg(cfg.EffectiveSubscriptionTopic())
	msg.Data = payload
	msg.Header.Set("Nats-Msg-Id", "integration-1")
	if _, err := js.PublishMsg(msg); err != nil {
		t.Fatalf("failed to publish test telegram: %v", err)
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, 20*time.Second)
	defer waitCancel()

	for {
		select {
		case <-waitCtx.Done():
			t.Fatalf("telegrams row not persisted: %v", waitCtx.Err())
		default:
		}

		var status string
		err := pool.QueryRow(ctx, `
			SELECT status FROM aviation.telegrams WHERE message_id = $1 LIMIT 1
		`, "TMQ2526").Scan(&status)
		if err == nil {
			if status == string(dto.MessageStatusParsed) {
				return
			}
			t.Logf("message persisted with status=%s, waiting for parsed", status)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func startPostgres(ctx context.Context, t *testing.T) (tc.Container, string) {
	t.Helper()
	req := tc.ContainerRequest{
		Image:        "timescale/timescaledb:2.15.2-pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "aviation",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		).WithDeadline(2 * time.Minute),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to resolve postgres host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to resolve postgres port: %v", err)
	}

	url := fmt.Sprintf("postgres://postgres:postgres@%s:%s/aviation?sslmode=disable", host, port.Port())
	return container, url
}

func startNATS(ctx context.Context, t *testing.T) (tc.Container, string) {
	t.Helper()
	req := tc.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp"},
		Cmd:          []string{"-js", "--server_name=integration"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("4222/tcp"),
			wait.ForLog("Server is ready"),
		).WithDeadline(2 * time.Minute),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start nats container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to resolve nats host: %v", err)
	}
	port, err := container.MappedPort(ctx, "4222")
	if err != nil {
		t.Fatalf("failed to resolve nats port: %v", err)
	}

	return container, fmt.Sprintf("nats://%s:%s", host, port.Port())
}

func buildTestConfig(natsURL, pgURL string) *config.Config {
	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:      natsURL,
			Mode:     "jetstream",
			Stream:   "INTEGRATION_TELEGRAM",
			Consumer: "integration-consumer",
			StreamLimits: config.StreamLimitsConfig{
				MaxMsgs:  1000,
				MaxBytes: 67108864,
				MaxAge:   time.Hour,
				Discard:  "old",
				Storage:  "file",
				Replicas: 1,
			},
			ConsumerRules: config.ConsumerRulesConfig{
				MaxDeliver:    3,
				AckWait:       15 * time.Second,
				MaxAckPending: 128,
				DeliverPolicy: "all",
				ReplayPolicy:  "instant",
			},
		},
		Postgres: config.PostgresConfig{
			URL:      pgURL,
			MaxConns: 4,
			MinConns: 1,
		},
		App: config.AppConfig{
			BatchSize:       1,
			BatchTimeout:    time.Second,
			MonitorInterval: time.Second,
		},
		Log: config.LogConfig{
			Level:  "debug",
			Format: "json",
		},
		Publisher: config.PublisherConfig{
			Topic: "integration.telegram.json",
		},
		Subscription: config.SubscriptionConfig{
			Topic: "integration.telegram.serial",
		},
		Telemetry: config.TelemetryConfig{
			Enabled: false,
		},
		Timeouts: config.TimeoutsConfig{
			Server:        5 * time.Second,
			ReconnectWait: 2 * time.Second,
			Close:         5 * time.Second,
			AckWait:       15 * time.Second,
		},
	}

	// Monitoring server disabled for tests.
	cfg.Monitoring.Disabled = true

	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("invalid integration config: %v", err))
	}

	return cfg
}

func applyDDL(ctx context.Context, pool *pgxpool.Pool) error {
	ddlPath := filepath.Join("..", "..", "internal", "infra", "postgres", "telegrams.ddl")
	bytes, err := os.ReadFile(ddlPath)
	if err != nil {
		return fmt.Errorf("read ddl: %w", err)
	}
	_, err = pool.Exec(ctx, string(bytes))
	return err
}

func TestInsertOne_ConcurrentDedup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pgContainer, pgURL := startPostgres(ctx, t)
	defer func() {
		_ = pgContainer.Terminate(context.Background())
	}()

	natsContainer, natsURL := startNATS(ctx, t)
	defer func() {
		_ = natsContainer.Terminate(context.Background())
	}()

	cfg := buildTestConfig(natsURL, pgURL)
	logger, err := loginfra.ProvideLogger(cfg)
	if err != nil {
		t.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync()

	pool, err := postgresinfra.ProvideDB(cfg, logger)
	if err != nil {
		t.Fatalf("failed to init postgres: %v", err)
	}
	defer pool.Close()

	if err := applyDDL(ctx, pool); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	repo, err := postgresinfra.ProvideRepository(pool, logger)
	if err != nil {
		t.Fatalf("failed to init repository: %v", err)
	}

	// Concurrent inserts of the same business key
	const concurrency = 10
	errCh := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			msg := &dto.ParsedTelegram{
				Uuid:       uuid.NewString(),
				MessageID:  "CONCURRENT_TEST",
				DateTime:   "120000",
				Category:   "ARR",
				Content:    "ZCZC CONCURRENT_TEST 120000",
				Status:     dto.MessageStatusParsed,
				Parsed:     true,
				ReceivedAt: time.Now(),
				ParsedAt:   time.Now(),
			}
			errCh <- repo.InsertOne(ctx, msg)
		}()
	}

	// Collect errors
	for i := 0; i < concurrency; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent insert %d failed: %v", i, err)
		}
	}

	// Verify exactly 1 row in aviation.telegrams
	var telegramCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM aviation.telegrams WHERE message_id = $1
	`, "CONCURRENT_TEST").Scan(&telegramCount)
	if err != nil {
		t.Fatalf("failed to count telegrams: %v", err)
	}
	if telegramCount != 1 {
		t.Errorf("expected 1 telegram row, got %d", telegramCount)
	}

	// Verify exactly 1 row in aviation.telegram_keys
	var keyCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM aviation.telegram_keys WHERE message_id = $1
	`, "CONCURRENT_TEST").Scan(&keyCount)
	if err != nil {
		t.Fatalf("failed to count telegram_keys: %v", err)
	}
	if keyCount != 1 {
		t.Errorf("expected 1 telegram_keys row, got %d", keyCount)
	}
}
