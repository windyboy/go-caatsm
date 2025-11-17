package nats

import (
	"errors"
	"testing"
	"time"

	configpkg "caatsm/internal/infra/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nats-io/nats.go"
)

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

	Describe("normalizeConsumerConfig", func() {
		It("applies default values when config fields are empty", func() {
			cfg := &configpkg.Config{
				NATS: configpkg.NATSConfig{
					Mode: "",
				},
				App:      configpkg.AppConfig{},
				Timeouts: configpkg.TimeoutsConfig{},
			}

			normCfg := normalizeConsumerConfig(cfg)

			Expect(normCfg.consumerName).To(Equal("telegram-consumer"))
			Expect(normCfg.mode).To(Equal("jetstream"))
			Expect(normCfg.streamName).To(Equal("TELEGRAM"))
			Expect(normCfg.batchSize).To(Equal(50))
			Expect(normCfg.batchTimeout).To(Equal(2 * time.Second))
			Expect(normCfg.monitorInterval).To(Equal(30 * time.Second))
			Expect(normCfg.ackWait).To(Equal(30 * time.Second))
		})

		It("uses provided values when config fields are set", func() {
			cfg := &configpkg.Config{
				NATS: configpkg.NATSConfig{
					Consumer: "custom-consumer",
					Mode:     "core",
					Stream:   "CUSTOM_STREAM",
					ConsumerRules: configpkg.ConsumerRulesConfig{
						AckWait: 60 * time.Second,
					},
				},
				App: configpkg.AppConfig{
					BatchSize:       100,
					BatchTimeout:    5 * time.Second,
					MonitorInterval: 60 * time.Second,
				},
				Timeouts: configpkg.TimeoutsConfig{
					AckWait: 45 * time.Second,
				},
			}

			normCfg := normalizeConsumerConfig(cfg)

			Expect(normCfg.consumerName).To(Equal("custom-consumer"))
			Expect(normCfg.mode).To(Equal("core"))
			Expect(normCfg.streamName).To(Equal("CUSTOM_STREAM"))
			Expect(normCfg.batchSize).To(Equal(100))
			Expect(normCfg.batchTimeout).To(Equal(5 * time.Second))
			Expect(normCfg.monitorInterval).To(Equal(60 * time.Second))
			Expect(normCfg.ackWait).To(Equal(60 * time.Second)) // Uses ConsumerRules.AckWait
		})

		It("falls back to Timeouts.AckWait when ConsumerRules.AckWait is zero", func() {
			cfg := &configpkg.Config{
				NATS: configpkg.NATSConfig{
					ConsumerRules: configpkg.ConsumerRulesConfig{
						AckWait: 0,
					},
				},
				Timeouts: configpkg.TimeoutsConfig{
					AckWait: 45 * time.Second,
				},
			}

			normCfg := normalizeConsumerConfig(cfg)

			Expect(normCfg.ackWait).To(Equal(45 * time.Second))
		})

		It("sets dlqSubject when DLQ is enabled", func() {
			cfg := &configpkg.Config{
				DLQ: configpkg.DLQConfig{
					Enabled: true,
					Subject: "caatsm.dlq",
				},
			}

			normCfg := normalizeConsumerConfig(cfg)

			Expect(normCfg.dlqSubject).To(Equal("caatsm.dlq"))
		})

		It("clears dlqSubject when DLQ is disabled", func() {
			cfg := &configpkg.Config{
				DLQ: configpkg.DLQConfig{
					Enabled: false,
					Subject: "caatsm.dlq",
				},
			}

			normCfg := normalizeConsumerConfig(cfg)

			Expect(normCfg.dlqSubject).To(Equal(""))
		})
	})
})
