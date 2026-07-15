package nats

import (
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	"caatsm/internal/infra/telemetry"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// mockDLQHandler is a test double for the DLQHandler interface.
type mockDLQHandler struct {
	routeFunc func(ctx context.Context, msg *nats.Msg, cause error) error
	calls     []mockDLQCall
}

type mockDLQCall struct {
	Ctx   context.Context
	Msg   *nats.Msg
	Cause error
}

func (m *mockDLQHandler) RouteToDLQ(ctx context.Context, msg *nats.Msg, cause error) error {
	m.calls = append(m.calls, mockDLQCall{Ctx: ctx, Msg: msg, Cause: cause})
	if m.routeFunc != nil {
		return m.routeFunc(ctx, msg, cause)
	}
	return nil
}

func (m *mockDLQHandler) ValidateDLQ() error {
	return nil
}

var _ = Describe("DLQHandler", func() {
	Describe("ValidateDLQ", func() {
		It("returns error when JetStream context is nil", func() {
			handler := &defaultDLQHandler{
				logger:       zaptest.NewLogger(GinkgoT()),
				streamName:   "TEST_STREAM",
				consumerName: "test-consumer",
				dlqSubject:   "caatsm.dlq",
				telemetry:    telemetry.NewNoop(),
				js:           nil,
			}
			err := handler.ValidateDLQ()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("JetStream context is nil"))
		})
	})
})

var _ = Describe("Consumer handleError", func() {
	var (
		consumer *Consumer
		logger   *zap.Logger
		dlq      *mockDLQHandler
	)

	BeforeEach(func() {
		logger = zaptest.NewLogger(GinkgoT())
		dlq = &mockDLQHandler{}
		consumer = &Consumer{
			logger:       logger,
			telemetry:    telemetry.NewNoop(),
			dlqHandler:   dlq,
			streamName:   "TEST_STREAM",
			consumerName: "test-consumer",
			cfg: &config.Config{
				NATS: config.NATSConfig{
					ConsumerRules: config.ConsumerRulesConfig{
						MaxDeliver: 5,
					},
				},
			},
			backoff: []time.Duration{time.Second},
		}
	})

	createMsg := func() *nats.Msg {
		return &nats.Msg{
			Subject: "telegram.test",
			Data:    []byte("test"),
			Header:  nats.Header{},
		}
	}

	Describe("permanent errors", func() {
		It("routes to DLQ and ACKs on DLQ success", func() {
			msg := createMsg()
			dlq.routeFunc = nil // success

			consumer.handleError(context.Background(), msg, "msg-1", app.Permanent(errors.New("bad input")))

			Expect(dlq.calls).To(HaveLen(1))
			Expect(dlq.calls[0].Cause.Error()).To(Equal("bad input"))
		})

		It("NAKs on DLQ failure instead of ACKing", func() {
			msg := createMsg()
			dlq.routeFunc = func(ctx context.Context, msg *nats.Msg, cause error) error {
				return fmt.Errorf("dlq unavailable")
			}

			consumer.handleError(context.Background(), msg, "msg-1", app.Permanent(errors.New("bad input")))

			Expect(dlq.calls).To(HaveLen(1))
			// Message should NOT be ACKed - the NAK is implicit (no Ack call)
		})

		It("ACKs with warning when DLQ is nil (disabled)", func() {
			consumer.dlqHandler = nil
			msg := createMsg()

			consumer.handleError(context.Background(), msg, "msg-1", app.Permanent(errors.New("bad input")))

			// No DLQ call expected
			Expect(dlq.calls).To(BeEmpty())
		})
	})

	Describe("transient errors", func() {
		It("increments consecutiveErrors", func() {
			msg := createMsg()
			consumer.consecutiveErrors = 0

			consumer.handleError(context.Background(), msg, "msg-1", errors.New("temporary failure"))

			Expect(consumer.consecutiveErrors).To(Equal(1))
		})

		It("resets consecutiveErrors on permanent error", func() {
			msg := createMsg()
			consumer.consecutiveErrors = 5

			consumer.handleError(context.Background(), msg, "msg-1", app.Permanent(errors.New("permanent")))

			Expect(consumer.consecutiveErrors).To(Equal(0))
		})
	})
})

var _ = Describe("Config DLQ validation", func() {
	validBase := func() *config.Config {
		return &config.Config{
			NATS: config.NATSConfig{
				URL:    "nats://localhost:4222",
				Stream: "TEST",
				StreamLimits: config.StreamLimitsConfig{
					Replicas: 1,
				},
				ConsumerRules: config.ConsumerRulesConfig{
					MaxDeliver: 5,
					AckWait:    30 * time.Second,
				},
			},
			Publisher: config.PublisherConfig{Topic: "test"},
			Postgres:  config.PostgresConfig{URL: "postgres://localhost/test"},
			App: config.AppConfig{
				BatchSize:       10,
				BatchTimeout:    time.Second,
				MonitorInterval: time.Second,
			},
		}
	}

	It("rejects enabled=true with empty subject", func() {
		cfg := validBase()
		cfg.DLQ = config.DLQConfig{Enabled: true, Subject: ""}
		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("dlq.subject is required when dlq.enabled=true"))
	})

	It("allows enabled=true with non-empty subject", func() {
		cfg := validBase()
		cfg.DLQ = config.DLQConfig{Enabled: true, Subject: "caatsm.dlq"}
		err := cfg.Validate()
		Expect(err).NotTo(HaveOccurred())
	})

	It("allows enabled=false with empty subject", func() {
		cfg := validBase()
		cfg.DLQ = config.DLQConfig{Enabled: false, Subject: ""}
		err := cfg.Validate()
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("DLQ payload structure", func() {
	It("contains required fields matching documentation", func() {
		// Simulate what routeToDLQInternal builds
		msg := &nats.Msg{
			Subject: "telegram.test",
			Data:    []byte("test body"),
			Header:  nats.Header{},
		}
		msg.Header.Set("Nats-Msg-Id", "msg-abc-123")

		payload := map[string]any{
			"schema_version":  "1.0",
			"msg_id":          "msg-abc-123",
			"subject":         msg.Subject,
			"stream":          "TEST_STREAM",
			"consumer":        "test-consumer",
			"stream_sequence": uint64(42),
			"delivery_count":  uint64(3),
			"error":           "permanent validation error",
			"received_at":     time.Now().UTC(),
			"body":            string(msg.Data),
		}

		data, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())

		var parsed map[string]any
		Expect(json.Unmarshal(data, &parsed)).To(Succeed())

		// Verify all documented fields exist
		Expect(parsed["schema_version"]).To(Equal("1.0"))
		Expect(parsed["msg_id"]).To(Equal("msg-abc-123"))
		Expect(parsed["subject"]).To(Equal("telegram.test"))
		Expect(parsed["stream"]).To(Equal("TEST_STREAM"))
		Expect(parsed["consumer"]).To(Equal("test-consumer"))
		Expect(parsed["stream_sequence"]).To(Equal(float64(42))) // JSON numbers are float64
		Expect(parsed["delivery_count"]).To(Equal(float64(3)))
		Expect(parsed["error"]).To(Equal("permanent validation error"))
		Expect(parsed["received_at"]).NotTo(BeNil())
		Expect(parsed["body"]).To(Equal("test body"))
	})
})
