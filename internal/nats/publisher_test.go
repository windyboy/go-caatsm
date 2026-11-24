package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/pipeline"
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

var _ = Describe("Publisher", func() {
	It("treats nil message or empty topic as permanent errors", func() {
		js := &stubPubJS{}
		pub, err := NewPublisher(js, &config.Config{Publisher: config.PublisherConfig{Topic: ""}}, zap.NewNop())
		Expect(err).NotTo(HaveOccurred())

		errNil := pub.Publish(context.Background(), nil)
		Expect(pipeline.IsPermanent(errNil)).To(BeTrue())

		errTopic := pub.Publish(context.Background(), &telegram.Message{UUID: "u-1"})
		Expect(pipeline.IsPermanent(errTopic)).To(BeTrue())
	})

	It("sets Nats-Msg-Id and distinguishes transient vs permanent JetStream publish errors", func() {
		js := &stubPubJS{}
		pub, err := NewPublisher(js, &config.Config{Publisher: config.PublisherConfig{Topic: "telegram.json"}}, zap.NewNop())
		Expect(err).NotTo(HaveOccurred())

		msg := &telegram.Message{UUID: "uuid-123", Category: "ARR"}
		Expect(pub.Publish(context.Background(), msg)).To(Succeed())
		Expect(js.lastMsg.Subject).To(Equal("telegram.json"))
		Expect(js.lastMsg.Header.Get("Nats-Msg-Id")).To(Equal("uuid-123"))

		js.err = nats.ErrNoResponders
		errTransient := pub.Publish(context.Background(), msg)
		Expect(errTransient).To(HaveOccurred())
		Expect(pipeline.IsPermanent(errTransient)).To(BeFalse())

		js.err = errors.New("permission violation")
		errPerm := pub.Publish(context.Background(), msg)
		Expect(errPerm).To(HaveOccurred())
		Expect(pipeline.IsPermanent(errPerm)).To(BeTrue())
	})
})
