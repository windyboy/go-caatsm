//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"caatsm/internal/config"
	natsinfra "caatsm/internal/nats"
	"caatsm/internal/observability"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type noopHandler struct{}

func (noopHandler) Handle(context.Context, []byte, string) error { return nil }

func TestNATSConsumerRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	natsContainer, natsURL := startNATS(ctx, t)
	defer func() {
		_ = natsContainer.Terminate(context.Background())
	}()

	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("jetstream context: %v", err)
	}

	autoProvision := true
	cfg := &config.Config{
		NATS: config.NATSConfig{
			URL:           natsURL,
			Mode:          "jetstream",
			Stream:        "RECOVERY_TELEGRAM",
			Consumer:      "recovery-consumer",
			AutoProvision: &autoProvision,
			StreamLimits: config.StreamLimitsConfig{
				MaxMsgs:  1000,
				MaxBytes: 64 * 1024 * 1024,
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
		Subscription: config.SubscriptionConfig{Topic: "recovery.telegram.serial"},
		Publisher:    config.PublisherConfig{Topic: "recovery.telegram.json"},
		DLQ:          config.DLQConfig{Enabled: true, Subject: "recovery.dlq"},
		App:          config.AppConfig{BatchSize: 1, BatchTimeout: time.Second, MonitorInterval: time.Second},
	}

	if _, err := natsinfra.NewConsumer(nc, js, noopHandler{}, cfg, observability.NewNoop(), zap.NewNop()); err != nil {
		t.Fatalf("initial NewConsumer failed: %v", err)
	}

	for _, name := range []string{"RECOVERY_TELEGRAM", "RECOVERY_TELEGRAM_OUT", "RECOVERY_TELEGRAM_DLQ"} {
		if _, err := js.StreamInfo(name); err != nil {
			t.Fatalf("expected stream %s to exist: %v", name, err)
		}
	}

	if err := js.DeleteConsumer("RECOVERY_TELEGRAM", "recovery-consumer"); err != nil && !errors.Is(err, nats.ErrConsumerNotFound) {
		t.Fatalf("delete consumer: %v", err)
	}

	if _, err := natsinfra.NewConsumer(nc, js, noopHandler{}, cfg, observability.NewNoop(), zap.NewNop()); err != nil {
		t.Fatalf("recovery NewConsumer failed: %v", err)
	}

	if _, err := js.ConsumerInfo("RECOVERY_TELEGRAM", "recovery-consumer"); err != nil {
		t.Fatalf("expected recovered consumer to exist: %v", err)
	}
}
