//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"caatsm/internal/config"
	natsinfra "caatsm/internal/infrastructure/nats"
	"caatsm/internal/repository/postgres"
	"caatsm/internal/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

const (
	defaultDatabaseURL = "postgres://postgres:postgres@localhost:55432/aviation?sslmode=disable"
	defaultNATSURL     = "nats://localhost:54222"
	streamName         = "INTEGRATION_TELEGRAMS"
	publishSubject     = "integration.telegrams"
)

const sampleARRMessage = `ZCZC TMQ2530 141614
GG ZBTJZXZX
141614 ZSHCZTZX
(ARR-CES5470-ZBTJ-ZSHC1614)
NNNN`

func TestMessageFlowEndToEnd(t *testing.T) {
	ctx := context.Background()

	pool := connectDatabase(t, ctx)
	t.Cleanup(pool.Close)

	applyMigrations(t, ctx, pool)
	truncateTelegrams(t, ctx, pool)

	nc, js := connectNATS(t)
	defer nc.Drain()

	ensureStream(t, js, publishSubject)
	subscribeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	subscription := subscribeToSubject(t, js, publishSubject)
	t.Cleanup(func() {
		subscription.Unsubscribe()
	})

	cfg := &config.Config{
		Publisher: config.PublisherConfig{Topic: publishSubject},
	}

	repo := postgres.NewTelegramRepository(pool)
	publisher, err := natsinfra.NewPublisher(js, cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}

	messageID := uuid.NewString()
	svc := service.NewMessageService(cfg, repo, publisher)
	if err := svc.ProcessMessage(subscribeCtx, []byte(sampleARRMessage), messageID); err != nil {
		t.Fatalf("failed to process message: %v", err)
	}

	stored, err := repo.GetByID(subscribeCtx, messageID)
	if err != nil {
		t.Fatalf("failed to fetch stored message: %v", err)
	}
	if stored == nil || stored.Category != "ARR" {
		t.Fatalf("unexpected stored message: %+v", stored)
	}

	msg := nextJetStreamMessage(t, subscription)
	if msg == nil {
		t.Fatalf("expected a published message but none received")
	}
	_ = msg.Ack()
}

func TestPublishFailsWhenConnectionUnavailable(t *testing.T) {
	ctx := context.Background()

	pool := connectDatabase(t, ctx)
	t.Cleanup(pool.Close)
	applyMigrations(t, ctx, pool)
	truncateTelegrams(t, ctx, pool)

	nc, js := connectNATS(t)
	defer nc.Close()
	ensureStream(t, js, publishSubject)

	cfg := &config.Config{
		Publisher: config.PublisherConfig{Topic: publishSubject},
	}
	repo := postgres.NewTelegramRepository(pool)
	publisher, err := natsinfra.NewPublisher(js, cfg)
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}

	// Close the NATS connection to simulate an outage.
	nc.Close()

	svc := service.NewMessageService(cfg, repo, publisher)
	err = svc.ProcessMessage(ctx, []byte(sampleARRMessage), uuid.NewString())
	if err == nil || !strings.Contains(err.Error(), "publish error") {
		t.Fatalf("expected publish error when NATS connection is closed, got %v", err)
	}
}

func connectDatabase(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	dbURL := getEnv("INTEGRATION_DATABASE_URL", defaultDatabaseURL)
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	return pool
}

func connectNATS(t *testing.T) (*nats.Conn, nats.JetStreamContext) {
	t.Helper()
	natsURL := getEnv("INTEGRATION_NATS_URL", defaultNATSURL)
	nc, err := nats.Connect(natsURL, nats.Timeout(5*time.Second))
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("failed to initialize JetStream: %v", err)
	}

	return nc, js
}

func ensureStream(t *testing.T, js nats.JetStreamContext, subject string) {
	t.Helper()
	if _, err := js.StreamInfo(streamName); err == nil {
		_ = js.DeleteStream(streamName)
	}

	_, err := js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
		Storage:  nats.FileStorage,
		Replicas: 1,
	})
	if err != nil {
		t.Fatalf("failed to configure stream: %v", err)
	}

	t.Cleanup(func() {
		_ = js.DeleteStream(streamName)
	})
}

func applyMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	migrations := []string{
		filepath.Join("..", "..", "migrations", "001_create_hypertable.up.sql"),
		filepath.Join("..", "..", "migrations", "002_add_continuous_aggregate.up.sql"),
	}

	for _, file := range migrations {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", file, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			t.Fatalf("failed to apply migration %s: %v", file, err)
		}
	}
}

func truncateTelegrams(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, "TRUNCATE TABLE aviation.telegrams"); err != nil {
		t.Fatalf("failed to truncate telegrams table: %v", err)
	}
}

func subscribeToSubject(t *testing.T, js nats.JetStreamContext, subject string) *nats.Subscription {
	t.Helper()
	sub, err := js.SubscribeSync(subject)
	if err != nil {
		t.Fatalf("failed to subscribe to subject %s: %v", subject, err)
	}
	return sub
}

func nextJetStreamMessage(t *testing.T, sub *nats.Subscription) *nats.Msg {
	t.Helper()
	msg, err := sub.NextMsg(5 * time.Second)
	if err != nil {
		t.Fatalf("failed to read jetstream message: %v", err)
	}
	return msg
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

