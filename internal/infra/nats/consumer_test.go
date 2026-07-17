package nats

import (
	"caatsm/internal/infra/config"
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type lifecycleMessage struct {
	metadata *nats.MsgMetadata
	ackCount int
	nakCount int
	delay    time.Duration
}

func (m *lifecycleMessage) Ack(...nats.AckOpt) error { m.ackCount++; return nil }
func (m *lifecycleMessage) Nak(...nats.AckOpt) error { m.nakCount++; return nil }
func (m *lifecycleMessage) NakWithDelay(delay time.Duration, _ ...nats.AckOpt) error {
	m.nakCount++
	m.delay = delay
	return nil
}
func (m *lifecycleMessage) Metadata() (*nats.MsgMetadata, error) { return m.metadata, nil }

func testConsumer(rec *mockTelemetryRecorder) *Consumer {
	return &Consumer{
		cfg: &config.Config{NATS: config.NATSConfig{ConsumerRules: config.ConsumerRulesConfig{
			MaxDeliver: 3,
			Backoff:    []time.Duration{time.Second},
		}}},
		logger:       zap.NewNop(),
		telemetry:    rec,
		streamName:   "TEST",
		consumerName: "consumer",
		backoff:      []time.Duration{time.Second},
	}
}

func TestNATS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NATS Suite")
}

var _ = Describe("Consumer helpers", func() {
	Describe("isJetStreamResourceNotFound", func() {
		It("detects missing resources for known errors", func() {
			Expect(isJetStreamResourceNotFound(nil)).To(BeFalse())
			Expect(isJetStreamResourceNotFound(nats.ErrStreamNotFound)).To(BeTrue())
			Expect(isJetStreamResourceNotFound(errors.New("consumer not found in stream not found"))).To(BeTrue())
		})
	})

	Describe("policy mapping", func() {
		It("maps deliver policies to NATS constants", func() {
			tests := map[string]nats.DeliverPolicy{
				"":                 nats.DeliverAllPolicy,
				"new":              nats.DeliverNewPolicy,
				"LAST":             nats.DeliverLastPolicy,
				"last_per_subject": nats.DeliverLastPerSubjectPolicy,
				"sequence":         nats.DeliverByStartSequencePolicy,
				"time":             nats.DeliverByStartTimePolicy,
				"unknown-so-far":   nats.DeliverAllPolicy,
			}
			for input, want := range tests {
				Expect(mapDeliverPolicy(input)).To(Equal(want))
			}
		})

		It("maps replay policy to instant by default", func() {
			Expect(mapReplayPolicy("original")).To(Equal(nats.ReplayOriginalPolicy))
			Expect(mapReplayPolicy("")).To(Equal(nats.ReplayInstantPolicy))
		})
	})

	Describe("DLQ terminal handling", func() {
		It("ACKs only after a successful DLQ publish", func() {
			consumer := testConsumer(&mockTelemetryRecorder{})
			msg := &lifecycleMessage{metadata: &nats.MsgMetadata{NumDelivered: 1}}

			consumer.completeDLQRouting(context.Background(), msg, "id", "permanent", nil)

			Expect(msg.ackCount).To(Equal(1))
			Expect(msg.nakCount).To(Equal(0))
		})

		It("delays NAK when DLQ publish fails before MaxDeliver", func() {
			consumer := testConsumer(&mockTelemetryRecorder{})
			msg := &lifecycleMessage{metadata: &nats.MsgMetadata{NumDelivered: 2}}

			consumer.completeDLQRouting(context.Background(), msg, "id", "permanent", errors.New("DLQ unavailable"))

			Expect(msg.ackCount).To(Equal(0))
			Expect(msg.nakCount).To(Equal(1))
			Expect(msg.delay).To(Equal(time.Second))
		})

		It("retains the original message and records a terminal failure at MaxDeliver", func() {
			recorder := &mockTelemetryRecorder{}
			consumer := testConsumer(recorder)
			msg := &lifecycleMessage{metadata: &nats.MsgMetadata{NumDelivered: 3}}

			consumer.completeDLQRouting(context.Background(), msg, "id", "max_deliver_exhausted", errors.New("DLQ unavailable"))

			Expect(msg.ackCount).To(Equal(0))
			Expect(msg.nakCount).To(Equal(0))
			Expect(recorder.dlqTerminalFailures).To(Equal(1))
		})
	})
})
