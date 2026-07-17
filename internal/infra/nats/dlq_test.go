package nats

import (
	"caatsm/internal/infra/config"
	"caatsm/internal/infra/telemetry"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// mockPublisher implements dlqPublisher for unit tests.
type mockPublisher struct {
	publishErr     error
	publishData    []byte
	publishSubject string
}

func (m *mockPublisher) Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	m.publishSubject = subj
	m.publishData = data
	if m.publishErr != nil {
		return nil, m.publishErr
	}
	return &nats.PubAck{Stream: "TEST", Sequence: 1}, nil
}

// mockTelemetryRecorder tracks calls for test assertions.
type mockTelemetryRecorder struct {
	dlqMessages         int
	dlqPublishFailures  int
	dlqTerminalFailures int
	dlqDisabled         int
}

func (m *mockTelemetryRecorder) RecordProcessingResult(ctx context.Context, status, category string, parseLatency time.Duration) {
}
func (m *mockTelemetryRecorder) RecordPublishFailure(ctx context.Context, category string) {}
func (m *mockTelemetryRecorder) RecordFailure(stage string)                                {}
func (m *mockTelemetryRecorder) RecordMessageHandled(ctx context.Context, stream, consumer, result string, elapsed time.Duration) {
}
func (m *mockTelemetryRecorder) RecordRetry(ctx context.Context, stream, consumer, reason string) {}
func (m *mockTelemetryRecorder) RecordDLQMessage(ctx context.Context, stream, consumer string) {
	m.dlqMessages++
}
func (m *mockTelemetryRecorder) RecordDLQPublishFailure(ctx context.Context, stream, consumer string) {
	m.dlqPublishFailures++
}
func (m *mockTelemetryRecorder) RecordDLQTerminalFailure(ctx context.Context, stream, consumer string) {
	m.dlqTerminalFailures++
}
func (m *mockTelemetryRecorder) RecordDLQDisabled(ctx context.Context, stream, consumer string) {
	m.dlqDisabled++
}
func (m *mockTelemetryRecorder) RecordJSAPICall(operation string)                                {}
func (m *mockTelemetryRecorder) RecordAFTNValidationError(ctx context.Context, errorType string) {}

var _ = Describe("DLQHandler", func() {
	var (
		handler *defaultDLQHandler
		logger  *zap.Logger
		mock    *mockPublisher
	)

	BeforeEach(func() {
		logger = zaptest.NewLogger(GinkgoT())
		mock = &mockPublisher{}
		handler = &defaultDLQHandler{
			publisher:    mock,
			logger:       logger,
			streamName:   "TEST_STREAM",
			consumerName: "test-consumer",
			dlqSubject:   "caatsm.dlq",
			telemetry:    telemetry.NewNoop(),
		}
	})

	Describe("RouteToDLQ", func() {
		It("publishes payload with expected fields on success", func() {
			msg := &nats.Msg{
				Subject: "telegram.test",
				Data:    []byte("test message body"),
				Header:  nats.Header{"Nats-Msg-Id": []string{"msg-123"}},
			}
			err := handler.RouteToDLQ(context.Background(), msg, errors.New("permanent error"))
			Expect(err).NotTo(HaveOccurred())

			Expect(mock.publishSubject).To(Equal("caatsm.dlq"))
			Expect(mock.publishData).NotTo(BeNil())

			var payload map[string]any
			Expect(json.Unmarshal(mock.publishData, &payload)).To(Succeed())
			Expect(payload["subject"]).To(Equal("telegram.test"))
			Expect(payload["stream"]).To(Equal("TEST_STREAM"))
			Expect(payload["consumer"]).To(Equal("test-consumer"))
			Expect(payload["error"]).To(Equal("permanent error"))
			Expect(payload["schema_version"]).To(Equal("v1"))
			Expect(payload["message_id"]).To(Equal("msg-123"))
			Expect(payload["body_base64"]).To(Equal(base64.StdEncoding.EncodeToString([]byte("test message body"))))
			Expect(payload["received_at"]).NotTo(BeNil())
		})

		It("includes headers in payload when present", func() {
			msg := &nats.Msg{
				Subject: "test",
				Data:    []byte("data"),
				Header:  nats.Header{"X-Custom": []string{"val1"}},
			}
			_ = handler.RouteToDLQ(context.Background(), msg, errors.New("err"))

			var payload map[string]any
			Expect(json.Unmarshal(mock.publishData, &payload)).To(Succeed())
			headers, ok := payload["headers"].(map[string]any)
			Expect(ok).To(BeTrue())
			vals, ok := headers["X-Custom"].([]any)
			Expect(ok).To(BeTrue())
			Expect(vals).To(ConsistOf("val1"))
		})

		It("returns error when publish fails", func() {
			mock.publishErr = errors.New("nats unavailable")
			msg := &nats.Msg{Subject: "test", Data: []byte("data")}
			err := handler.RouteToDLQ(context.Background(), msg, errors.New("perm"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("nats unavailable"))
		})

		It("records telemetry on successful publish", func() {
			rec := &mockTelemetryRecorder{}
			handler.telemetry = rec
			msg := &nats.Msg{Subject: "test", Data: []byte("data")}
			_ = handler.RouteToDLQ(context.Background(), msg, errors.New("perm"))
			Expect(rec.dlqMessages).To(Equal(1))
			Expect(rec.dlqPublishFailures).To(Equal(0))
		})

		It("records telemetry on publish failure", func() {
			rec := &mockTelemetryRecorder{}
			handler.telemetry = rec
			mock.publishErr = errors.New("fail")
			msg := &nats.Msg{Subject: "test", Data: []byte("data")}
			_ = handler.RouteToDLQ(context.Background(), msg, errors.New("perm"))
			Expect(rec.dlqMessages).To(Equal(0))
			Expect(rec.dlqPublishFailures).To(Equal(1))
		})
	})
})

var _ = Describe("Config DLQ validation", func() {
	It("rejects dlq.enabled=true with empty subject", func() {
		cfg := &config.Config{
			NATS:      config.NATSConfig{URL: "nats://localhost", Stream: "S", Consumer: "C"},
			App:       config.AppConfig{BatchSize: 1, BatchTimeout: time.Second, MonitorInterval: time.Second},
			Publisher: config.PublisherConfig{Topic: "test.topic"},
			Postgres:  config.PostgresConfig{URL: "postgres://localhost"},
			DLQ:       config.DLQConfig{Enabled: true, Subject: ""},
		}
		cfg.NATS.StreamLimits.Replicas = 1
		cfg.NATS.ConsumerRules.MaxDeliver = 5
		cfg.NATS.ConsumerRules.AckWait = 30 * time.Second
		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("dlq.subject is required when dlq.enabled=true"))
	})

	It("allows dlq.enabled=true with non-empty subject", func() {
		cfg := &config.Config{
			NATS:      config.NATSConfig{URL: "nats://localhost", Stream: "S", Consumer: "C"},
			App:       config.AppConfig{BatchSize: 1, BatchTimeout: time.Second, MonitorInterval: time.Second},
			Publisher: config.PublisherConfig{Topic: "test.topic"},
			Postgres:  config.PostgresConfig{URL: "postgres://localhost"},
			DLQ:       config.DLQConfig{Enabled: true, Subject: "caatsm.dlq"},
		}
		cfg.NATS.StreamLimits.Replicas = 1
		cfg.NATS.ConsumerRules.MaxDeliver = 5
		cfg.NATS.ConsumerRules.AckWait = 30 * time.Second
		err := cfg.Validate()
		Expect(err).NotTo(HaveOccurred())
	})

	It("allows dlq.enabled=false with empty subject", func() {
		cfg := &config.Config{
			NATS:      config.NATSConfig{URL: "nats://localhost", Stream: "S", Consumer: "C"},
			App:       config.AppConfig{BatchSize: 1, BatchTimeout: time.Second, MonitorInterval: time.Second},
			Publisher: config.PublisherConfig{Topic: "test.topic"},
			Postgres:  config.PostgresConfig{URL: "postgres://localhost"},
			DLQ:       config.DLQConfig{Enabled: false, Subject: ""},
		}
		cfg.NATS.StreamLimits.Replicas = 1
		cfg.NATS.ConsumerRules.MaxDeliver = 5
		cfg.NATS.ConsumerRules.AckWait = 30 * time.Second
		err := cfg.Validate()
		Expect(err).NotTo(HaveOccurred())
	})
})
