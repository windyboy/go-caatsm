package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/telegram"
	"context"
	"errors"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

type stubPubJS struct {
	nats.JetStreamContext
	lastMsg *nats.Msg
	err     error
}

func (s *stubPubJS) PublishMsg(m *nats.Msg, _ ...nats.PubOpt) (*nats.PubAck, error) {
	s.lastMsg = m
	if s.err != nil {
		return nil, s.err
	}
	return &nats.PubAck{Stream: "TELEGRAM_OUT", Sequence: 1}, nil
}

type spyPubRecorder struct {
	ops []string
}

func (s *spyPubRecorder) RecordJSAPICall(op string) {
	s.ops = append(s.ops, op)
}

var _ = Describe("Publisher", func() {
	It("treats nil message or empty topic as permanent errors", func() {
		js := &stubPubJS{}
		pub, err := NewPublisher(js, &config.Config{Publisher: config.PublisherConfig{Topic: ""}}, nil, zap.NewNop())
		Expect(err).NotTo(HaveOccurred())

		errNil := pub.Publish(context.Background(), nil)
		Expect(isPermanentError(errNil)).To(BeTrue())

		errTopic := pub.Publish(context.Background(), &telegram.Message{UUID: "u-1"})
		Expect(isPermanentError(errTopic)).To(BeTrue())
	})

	It("sets Nats-Msg-Id, records JS API calls, and distinguishes transient vs permanent errors", func() {
		js := &stubPubJS{}
		rec := &spyPubRecorder{}
		pub, err := NewPublisher(js, &config.Config{Publisher: config.PublisherConfig{Topic: "telegram.json"}}, rec, zap.NewNop())
		Expect(err).NotTo(HaveOccurred())

		msg := &telegram.Message{UUID: "uuid-123", Category: "ARR"}
		Expect(pub.Publish(context.Background(), msg)).To(Succeed())
		Expect(rec.ops).To(Equal([]string{"publish"}))
		Expect(js.lastMsg.Subject).To(Equal("telegram.json"))
		Expect(js.lastMsg.Header.Get("Nats-Msg-Id")).To(Equal("uuid-123"))

		js.err = nats.ErrNoResponders
		errTransient := pub.Publish(context.Background(), msg)
		Expect(errTransient).To(HaveOccurred())
		Expect(isPermanentError(errTransient)).To(BeFalse())

		js.err = errors.New("permission violation")
		errPerm := pub.Publish(context.Background(), msg)
		Expect(errPerm).To(HaveOccurred())
		Expect(isPermanentError(errPerm)).To(BeTrue())
	})
})
