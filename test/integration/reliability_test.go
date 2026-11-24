//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"caatsm/internal/config"
	natsinfra "caatsm/internal/nats"
	"caatsm/internal/observability"
	"caatsm/internal/pipeline"
	"caatsm/internal/postgres"
	"caatsm/internal/telegram"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type timeoutHandler struct {
	attempts atomic.Int32
}

func (h *timeoutHandler) Handle(ctx context.Context, _ []byte, _ string) error {
	h.attempts.Add(1)
	<-ctx.Done()
	return ctx.Err()
}

type noopStore struct{}

func (noopStore) InsertOne(context.Context, *telegram.Message) error { return nil }
func (noopStore) InsertRaw(context.Context, *telegram.Message) error { return nil }

func TestReliabilityAndMigrationScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	pgContainer, pgURL := startPostgres(ctx, t)
	defer func() {
		_ = pgContainer.Terminate(context.Background())
	}()

	natsContainer, natsURL := startNATS(ctx, t)
	defer func() {
		_ = natsContainer.Terminate(context.Background())
	}()

	t.Run("PopulatedTelegramsRawHypertableMigration", func(t *testing.T) {
		cfg := buildTestConfig(natsURL, pgURL)
		pool, err := postgres.OpenPool(ctx, cfg, zap.NewNop())
		if err != nil {
			t.Fatalf("open pool: %v", err)
		}
		defer pool.Close()

		// Simulate pre-existing schema where aviation.telegrams_raw is a populated regular table.
		setupSQL := `
			CREATE SCHEMA IF NOT EXISTS aviation;
			CREATE EXTENSION IF NOT EXISTS timescaledb;
			CREATE TABLE IF NOT EXISTS aviation.telegrams_raw (
				uuid UUID NOT NULL,
				status TEXT NOT NULL,
				error_reason TEXT,
				content TEXT NOT NULL,
				received_at TIMESTAMPTZ NOT NULL,
				metadata JSONB,
				PRIMARY KEY (uuid, received_at)
			);
		`
		if _, err := pool.Exec(ctx, setupSQL); err != nil {
			t.Fatalf("setup legacy raw table: %v", err)
		}

		seedID := uuid.New()
		if _, err := pool.Exec(ctx, `
			INSERT INTO aviation.telegrams_raw (uuid, status, error_reason, content, received_at, metadata)
			VALUES ($1, 'body_error', 'legacy error', 'LEGACY RAW CONTENT', NOW(), '{}'::jsonb)
		`, seedID); err != nil {
			t.Fatalf("seed legacy raw row: %v", err)
		}

		// Apply updated DDL twice to verify migration + idempotency.
		if err := applyDDL(ctx, pool); err != nil {
			t.Fatalf("applyDDL on populated raw table failed: %v", err)
		}
		if err := applyDDL(ctx, pool); err != nil {
			t.Fatalf("re-applying DDL failed: %v", err)
		}

		var count int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM aviation.telegrams_raw WHERE uuid = $1`, seedID).Scan(&count); err != nil {
			t.Fatalf("query migrated row: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 preserved raw row, got %d", count)
		}

		var isHypertable bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM timescaledb_information.hypertables
				WHERE hypertable_schema = 'aviation' AND hypertable_name = 'telegrams_raw'
			)
		`).Scan(&isHypertable); err != nil {
			t.Fatalf("check hypertable catalog: %v", err)
		}
		if !isHypertable {
			t.Fatalf("expected aviation.telegrams_raw to be converted to a hypertable")
		}
	})

	t.Run("LegacyWeatherReplayAndDistinctReports", func(t *testing.T) {
		cfg := buildTestConfig(natsURL, pgURL)
		pool, err := postgres.OpenPool(ctx, cfg, zap.NewNop())
		if err != nil {
			t.Fatalf("open pool: %v", err)
		}
		defer pool.Close()

		if err := applyDDL(ctx, pool); err != nil {
			t.Fatalf("applyDDL: %v", err)
		}

		router := telegram.NewRouter()
		repo := postgres.NewRepository(pool, observability.NewNoop(), zap.NewNop())

		rawMetar := "METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013="
		parsedMetar, err := router.Parse(rawMetar)
		if err != nil {
			t.Fatalf("parse metar: %v", err)
		}

		// Seed legacy row using old {station}-{issue_time} message_id format.
		legacyID := fmt.Sprintf("KJFK-%s", parsedMetar.MessageID[len("METAR-KJFK-"):])
		if _, err := pool.Exec(ctx, `
			INSERT INTO aviation.telegrams (uuid, message_id, date_time, category, content, status, received_at)
			VALUES ($1, $2, $3, 'METAR', $4, 'parsed', NOW())
		`, uuid.New(), legacyID, parsedMetar.DateTime, rawMetar); err != nil {
			t.Fatalf("seed legacy metar row: %v", err)
		}

		// Replaying the same METAR with the new METAR-KJFK-... key must be skipped.
		if err := repo.InsertOne(ctx, parsedMetar); err != nil {
			t.Fatalf("replay metar: %v", err)
		}

		// Same station and issue time TAF, COR, and a second distinct COR must all persist.
		distinctPayloads := []string{
			"TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020=",
			"METAR COR KJFK 251200Z 36015KT 10SM FEW020 25/18 Q1013=",
			"METAR COR KJFK 251200Z 01018KT 10SM FEW020 25/18 Q1013=",
		}
		for _, raw := range distinctPayloads {
			msg, parseErr := router.Parse(raw)
			if parseErr != nil {
				t.Fatalf("parse %q: %v", raw, parseErr)
			}
			if err := repo.InsertOne(ctx, msg); err != nil {
				t.Fatalf("insert %q: %v", raw, err)
			}
		}

		var total int
		if err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM aviation.telegrams WHERE date_time = $1 AND category IN ('METAR', 'TAF')
		`, parsedMetar.DateTime).Scan(&total); err != nil {
			t.Fatalf("count weather rows: %v", err)
		}
		if total != 4 {
			t.Fatalf("expected 4 weather rows (1 legacy METAR + 1 TAF + 2 distinct METAR COR), got %d", total)
		}
	})

	t.Run("FinalDeliveryTimeoutRoutesToDLQ", func(t *testing.T) {
		nc, err := nats.Connect(natsURL)
		if err != nil {
			t.Fatalf("connect nats: %v", err)
		}
		defer nc.Close()

		js, err := nc.JetStream()
		if err != nil {
			t.Fatalf("jetstream: %v", err)
		}

		auto := true
		cfg := buildTestConfig(natsURL, pgURL)
		cfg.NATS.Stream = "TIMEOUT_DLQ_IN"
		cfg.NATS.PublishStream = "TIMEOUT_DLQ_OUT"
		cfg.NATS.DLQStream = "TIMEOUT_DLQ_DLQ"
		cfg.NATS.Consumer = "timeout-dlq-consumer"
		cfg.NATS.AutoProvision = &auto
		cfg.NATS.ConsumerRules.MaxDeliver = 2
		cfg.NATS.ConsumerRules.Backoff = []time.Duration{50 * time.Millisecond}
		cfg.Subscription.Topic = "timeout.serial"
		cfg.Publisher.Topic = "timeout.json"
		cfg.DLQ = config.DLQConfig{Enabled: true, Subject: "timeout.dlq"}
		cfg.Timeouts.Close = 200 * time.Millisecond
		cfg.App.BatchTimeout = 200 * time.Millisecond

		h := &timeoutHandler{}
		consumer, err := natsinfra.NewConsumer(nc, js, h, cfg, observability.NewNoop(), zap.NewNop())
		if err != nil {
			t.Fatalf("new consumer: %v", err)
		}

		runCtx, runCancel := context.WithCancel(ctx)
		errCh := make(chan error, 1)
		go func() { errCh <- consumer.Start(runCtx) }()
		defer func() {
			runCancel()
			<-errCh
		}()

		if _, err := js.Publish(cfg.Subscription.Topic, []byte("slow-telegram")); err != nil {
			t.Fatalf("publish inbound: %v", err)
		}

		dlqSub, err := js.PullSubscribe(cfg.DLQ.Subject, "dlq-verifier", nats.BindStream(cfg.NATS.DLQStream))
		if err != nil {
			t.Fatalf("subscribe dlq: %v", err)
		}
		defer func() { _ = dlqSub.Unsubscribe() }()

		msgs, err := dlqSub.Fetch(1, nats.MaxWait(10*time.Second))
		if err != nil {
			t.Fatalf("expected dead-letter message after final timeout delivery: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("expected 1 DLQ message, got %d", len(msgs))
		}
		_ = msgs[0].Ack()

		var dlqBody map[string]any
		if err := json.Unmarshal(msgs[0].Data, &dlqBody); err != nil {
			t.Fatalf("unmarshal dlq payload: %v", err)
		}
		if dlqBody["body"] != "slow-telegram" {
			t.Fatalf("unexpected dlq body: %v", dlqBody["body"])
		}

		// Original message must have been ACKed from the WorkQueue inbound stream.
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			info, infoErr := js.StreamInfo(cfg.NATS.Stream)
			if infoErr == nil && info.State.Msgs == 0 {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		info, _ := js.StreamInfo(cfg.NATS.Stream)
		t.Fatalf("expected inbound stream to be drained after DLQ ACK, remaining msgs=%d", info.State.Msgs)
	})

	t.Run("SharedUpstreamIDPublishesBothRecords", func(t *testing.T) {
		nc, err := nats.Connect(natsURL)
		if err != nil {
			t.Fatalf("connect nats: %v", err)
		}
		defer nc.Close()

		js, err := nc.JetStream()
		if err != nil {
			t.Fatalf("jetstream: %v", err)
		}

		auto := true
		cfg := buildTestConfig(natsURL, pgURL)
		cfg.NATS.Stream = "DEDUP_TEST_IN"
		cfg.NATS.PublishStream = "DEDUP_TEST_OUT"
		cfg.NATS.Consumer = "dedup-test-consumer"
		cfg.NATS.AutoProvision = &auto
		cfg.Subscription.Topic = "dedup.serial"
		cfg.Publisher.Topic = "dedup.json"
		cfg.App.BatchTimeout = 200 * time.Millisecond

		// Create inbound stream with short duplicate window so two distinct records can share
		// the same upstream Nats-Msg-Id without being dropped at inbound ingest.
		if _, err := js.AddStream(&nats.StreamConfig{
			Name:       cfg.NATS.Stream,
			Subjects:   []string{cfg.Subscription.Topic},
			Retention:  nats.WorkQueuePolicy,
			Duplicates: 200 * time.Millisecond,
		}); err != nil {
			t.Fatalf("add inbound stream: %v", err)
		}

		pub, err := natsinfra.NewPublisher(js, cfg, observability.NewNoop(), zap.NewNop())
		if err != nil {
			t.Fatalf("new publisher: %v", err)
		}
		proc := pipeline.NewProcessor(
			telegram.NewRouter(),
			telegram.NewAFTNValidator(),
			noopStore{},
			pub,
			observability.NewNoop(),
			zap.NewNop(),
			pipeline.Options{},
		)
		consumer, err := natsinfra.NewConsumer(nc, js, proc, cfg, observability.NewNoop(), zap.NewNop())
		if err != nil {
			t.Fatalf("new consumer: %v", err)
		}

		runCtx, runCancel := context.WithCancel(ctx)
		errCh := make(chan error, 1)
		go func() { errCh <- consumer.Start(runCtx) }()
		defer func() {
			runCancel()
			<-errCh
		}()

		const sharedID = "upstream-reused-id"
		msg1 := nats.NewMsg(cfg.Subscription.Topic)
		msg1.Data = []byte("METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=")
		msg1.Header.Set("Nats-Msg-Id", sharedID)
		if _, err := js.PublishMsg(msg1); err != nil {
			t.Fatalf("publish msg1: %v", err)
		}

		// Wait past inbound 200ms dedup window so JetStream stores msg2 as a second distinct record
		// with the same upstream Nats-Msg-Id (still well inside outbound stream's default 2m dedup window).
		time.Sleep(300 * time.Millisecond)

		msg2 := nats.NewMsg(cfg.Subscription.Topic)
		msg2.Data = []byte("METAR KORD 251200Z 27010KT 10SM SCT030 20/14 Q1015=")
		msg2.Header.Set("Nats-Msg-Id", sharedID)
		if _, err := js.PublishMsg(msg2); err != nil {
			t.Fatalf("publish msg2: %v", err)
		}

		outSub, err := js.PullSubscribe(cfg.Publisher.Topic, "out-verifier", nats.BindStream(cfg.NATS.PublishStream))
		if err != nil {
			t.Fatalf("subscribe outbound: %v", err)
		}
		defer func() { _ = outSub.Unsubscribe() }()

		var received []*nats.Msg
		deadline := time.Now().Add(10 * time.Second)
		for len(received) < 2 && time.Now().Before(deadline) {
			batch, fetchErr := outSub.Fetch(2-len(received), nats.MaxWait(500*time.Millisecond))
			if fetchErr != nil && !errors.Is(fetchErr, nats.ErrTimeout) {
				t.Fatalf("fetch outbound: %v", fetchErr)
			}
			for _, m := range batch {
				_ = m.Ack()
				received = append(received, m)
			}
		}

		if len(received) != 2 {
			t.Fatalf("expected 2 outbound messages despite shared upstream Nats-Msg-Id, got %d", len(received))
		}
		if received[0].Header.Get("Nats-Msg-Id") == received[1].Header.Get("Nats-Msg-Id") {
			t.Fatalf("expected distinct outbound Nats-Msg-Id values, got identical %q", received[0].Header.Get("Nats-Msg-Id"))
		}
	})
}
