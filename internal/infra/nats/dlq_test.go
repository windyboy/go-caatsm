package nats

import (
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

// mockPublisher implements dlqPublisher for unit tests.
type mockPublisher struct {
	publishErr     error
	publishData    []byte
	publishSubject string
	streamNames    map[string]nats.StreamInfo
	streamNameErr  error
}

func (m *mockPublisher) Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	m.publishSubject = subj
	m.publishData = data
	if m.publishErr != nil {
		return nil, m.publishErr
	}
	return &nats.PubAck{Stream: "TEST", Sequence: 1}, nil
}

func (m *mockPublisher) StreamNameBySubject(subj string, opts ...nats.JSOpt) (string, error) {
	if m.streamNameErr != nil {
		return "", m.streamNameErr
	}
	if m.streamNames != nil {
		if _, ok := m.streamNames[subj]; ok {
			return "TEST_STREAM", nil
		}
		return "", fmt.Errorf("subject not found")
	}
	return "TEST_STREAM", nil
}

// mockTelemetryRecorder tracks calls for test assertions.
type mockTelemetryRecorder struct {
	dlqMessages        int
	dlqPublishFailures int
}

func (m *mockTelemetryRecorder) RecordProcessingResult(ctx context.Context, status, category string, parseLatency time.Duration) {
}
func (m *mockTelemetryRecorder) RecordPublishFailure(ctx context.Context, category string)         {}
func (m *mockTelemetryRecorder) RecordFailure(stage string)                                         {}
func (m *mockTelemetryRecorder) RecordMessageHandled(ctx context.Context, stream, consumer, result string, elapsed time.Duration) {
}
func (m *mockTelemetryRecorder) RecordRetry(ctx context.Context, stream, consumer, reason string)  {}
func (m *mockTelemetryRecorder) RecordDLQMessage(ctx context.Context, stream, consumer string)      { m.dlqMessages++ }
func (m *mockTelemetryRecorder) RecordDLQPublishFailure(ctx context.Context, stream, consumer string) {
	m.dlqPublishFailures++
}
func (m *mockTelemetryRecorder) RecordJSAPICall(operation string)                                   {}
func (m *mockTelemetryRecorder) RecordAFTNValidationError(ctx context.Context, errorType string)     {}

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

	Describe("ValidateDLQ", func() {
		It("returns error when publisher is nil", func() {
			handler.publisher = nil
			err := handler.ValidateDLQ()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("JetStream context is nil"))
		})

		It("returns error when subject is not bound to any stream", func() {
			mock.streamNameErr = errors.New("no stream found")
			err := handler.ValidateDLQ()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not bound to any JetStream stream"))
		})

		It("succeeds when subject is bound to a stream", func() {
			mock.streamNames = map[string]nats.StreamInfo{
				"caatsm.dlq": {},
			}
			err := handler.ValidateDLQ()
			Expect(err).NotTo(HaveOccurred())
		})
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
			Expect(payload["body"]).To(Equal("test message body"))
			Expect(payload["received_at"]).NotTo(BeNil())
			Expect(payload["transport_msg_id"]).To(Equal("msg-123"))
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

		It("includes reply in payload when present", func() {
			msg := &nats.Msg{
				Subject: "test",
				Data:    []byte("data"),
				Reply:   "reply.subject",
			}
			_ = handler.RouteToDLQ(context.Background(), msg, errors.New("err"))

			var payload map[string]any
			Expect(json.Unmarshal(mock.publishData, &payload)).To(Succeed())
			Expect(payload["reply"]).To(Equal("reply.subject"))
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

var _ = Describe("Consumer DLQ handler initialization", func() {
	It("creates DLQ handler when enabled and subject non-empty", func() {
		cfg := &config.Config{
			DLQ: config.DLQConfig{Enabled: true, Subject: "caatsm.dlq"},
		}
		var handler DLQHandler
		if cfg.DLQ.Enabled && cfg.DLQ.Subject != "" {
			handler = &defaultDLQHandler{
				dlqSubject: cfg.DLQ.Subject,
				logger:     zaptest.NewLogger(GinkgoT()),
				telemetry:  telemetry.NewNoop(),
			}
		}
		Expect(handler).NotTo(BeNil())
	})

	It("does not create DLQ handler when subject is empty", func() {
		cfg := &config.Config{
			DLQ: config.DLQConfig{Enabled: true, Subject: ""},
		}
		var handler DLQHandler
		if cfg.DLQ.Enabled && cfg.DLQ.Subject != "" {
			handler = &defaultDLQHandler{
				dlqSubject: cfg.DLQ.Subject,
				logger:     zaptest.NewLogger(GinkgoT()),
				telemetry:  telemetry.NewNoop(),
			}
		}
		Expect(handler).To(BeNil())
	})

	It("does not create DLQ handler when disabled", func() {
		cfg := &config.Config{
			DLQ: config.DLQConfig{Enabled: false, Subject: "caatsm.dlq"},
		}
		var handler DLQHandler
		if cfg.DLQ.Enabled && cfg.DLQ.Subject != "" {
			handler = &defaultDLQHandler{
				dlqSubject: cfg.DLQ.Subject,
				logger:     zaptest.NewLogger(GinkgoT()),
				telemetry:  telemetry.NewNoop(),
			}
		}
		Expect(handler).To(BeNil())
	})
})
