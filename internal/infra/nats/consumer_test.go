package nats

import (
	"errors"
	"testing"

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
})