package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/telegram"
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

type fakeAckMsg struct {
	acked     bool
	nacked    bool
	nakDelay  time.Duration
	delivered uint64
	streamSeq uint64
}

func (f *fakeAckMsg) Ack(...nats.AckOpt) error { f.acked = true; return nil }
func (f *fakeAckMsg) Nak(...nats.AckOpt) error { f.nacked = true; return nil }
func (f *fakeAckMsg) NakWithDelay(d time.Duration, _ ...nats.AckOpt) error {
	f.nacked = true
	f.nakDelay = d
	return nil
}
func (f *fakeAckMsg) Metadata() (*nats.MsgMetadata, error) {
	return &nats.MsgMetadata{
		NumDelivered: f.delivered,
		Sequence:     nats.SequencePair{Stream: f.streamSeq},
	}, nil
}

type fakeDLQ struct {
	err        error
	called     bool
	lastCtxErr error
	hasDeadln  bool
}

func (d *fakeDLQ) RouteToDLQ(ctx context.Context, _ *nats.Msg, _ error) error {
	d.called = true
	d.lastCtxErr = ctx.Err()
	_, d.hasDeadln = ctx.Deadline()
	return d.err
}
func (d *fakeDLQ) ValidateDLQ(context.Context) error { return nil }

type spyRecorder struct {
	retries      []string
	handled      []string
	dlqCount     int
	dlqFailCount int
	jsCalls      int
	seqGaps      []uint64
}

func (s *spyRecorder) RecordMessageHandled(_ context.Context, _, _, result string, _ time.Duration) {
	s.handled = append(s.handled, result)
}
func (s *spyRecorder) RecordRetry(_ context.Context, _, _, reason string) {
	s.retries = append(s.retries, reason)
}
func (s *spyRecorder) RecordDLQMessage(context.Context, string, string)        { s.dlqCount++ }
func (s *spyRecorder) RecordDLQPublishFailure(context.Context, string, string) { s.dlqFailCount++ }
func (s *spyRecorder) RecordJSAPICall(string)                                  { s.jsCalls++ }
func (s *spyRecorder) RecordNATSConsumerPending(string, string, uint64)        {}
func (s *spyRecorder) RecordMessageGap(string, string, float64)                {}
func (s *spyRecorder) RecordSequenceGap(_ string, _ string, gap uint64) {
	s.seqGaps = append(s.seqGaps, gap)
}
func (s *spyRecorder) RecordSerialReaderHealth(string, string, bool) {}

type stubJSInfra struct {
	nats.JetStreamContext
	streams   map[string]*nats.StreamInfo
	consumers map[string]*nats.ConsumerInfo
	addedStr  []string
	addedCons []string
}

func (s *stubJSInfra) StreamNameBySubject(subject string, _ ...nats.JSOpt) (string, error) {
	for name, info := range s.streams {
		for _, subj := range info.Config.Subjects {
			if subj == subject {
				return name, nil
			}
		}
	}
	return "", nats.ErrStreamNotFound
}

func (s *stubJSInfra) StreamInfo(name string, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	if info, ok := s.streams[name]; ok {
		return info, nil
	}
	return nil, nats.ErrStreamNotFound
}

func (s *stubJSInfra) AddStream(cfg *nats.StreamConfig, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	s.addedStr = append(s.addedStr, cfg.Name)
	info := &nats.StreamInfo{Config: *cfg}
	s.streams[cfg.Name] = info
	return info, nil
}

func (s *stubJSInfra) ConsumerInfo(stream, consumer string, _ ...nats.JSOpt) (*nats.ConsumerInfo, error) {
	if info, ok := s.consumers[stream+"/"+consumer]; ok {
		return info, nil
	}
	return nil, nats.ErrConsumerNotFound
}

func (s *stubJSInfra) AddConsumer(stream string, cfg *nats.ConsumerConfig, _ ...nats.JSOpt) (*nats.ConsumerInfo, error) {
	s.addedCons = append(s.addedCons, stream+"/"+cfg.Durable)
	info := &nats.ConsumerInfo{Stream: stream, Name: cfg.Durable, Config: *cfg}
	s.consumers[stream+"/"+cfg.Durable] = info
	return info, nil
}

var _ = Describe("Consumer", func() {
	Describe("URL & Policy Helpers", func() {
		It("sanitizes URLs and maps JetStream policies", func() {
			Expect(sanitizeURLForLogging("nats://user:pass@localhost:4222")).To(Equal("nats://localhost:4222"))
			Expect(sanitizeURLForLogging("nats://localhost:4222")).To(Equal("nats://localhost:4222"))
			Expect(sanitizeURLForLogging("")).To(Equal(""))
			Expect(sanitizeURLForLogging("://invalid")).To(Equal("***"))

			Expect(isJetStreamResourceNotFound(nil)).To(BeFalse())
			Expect(isJetStreamResourceNotFound(nats.ErrStreamNotFound)).To(BeTrue())
			Expect(isJetStreamResourceNotFound(errors.New("consumer not found"))).To(BeTrue())

			Expect(mapDeliverPolicy("new")).To(Equal(nats.DeliverNewPolicy))
			Expect(mapDeliverPolicy(" new ")).To(Equal(nats.DeliverNewPolicy))
			Expect(mapDeliverPolicy("sequence")).To(Equal(nats.DeliverByStartSequencePolicy))
			Expect(mapDeliverPolicy("time")).To(Equal(nats.DeliverByStartTimePolicy))
			Expect(mapDeliverPolicy("")).To(Equal(nats.DeliverAllPolicy))
			Expect(mapReplayPolicy("original")).To(Equal(nats.ReplayOriginalPolicy))
			Expect(mapReplayPolicy(" original ")).To(Equal(nats.ReplayOriginalPolicy))
			Expect(mapReplayPolicy("")).To(Equal(nats.ReplayInstantPolicy))
		})
	})

	Describe("Infrastructure provisioning", func() {
		It("creates separated inbound/outbound/dlq streams when auto_provision is enabled", func() {
			js := &stubJSInfra{
				streams:   map[string]*nats.StreamInfo{},
				consumers: map[string]*nats.ConsumerInfo{},
			}
			auto := true
			cfg := &config.Config{
				NATS: config.NATSConfig{
					Stream:        "TELEGRAM",
					Consumer:      "c1",
					AutoProvision: &auto,
					ConsumerRules: config.ConsumerRulesConfig{
						DeliverPolicy: "sequence",
						StartSequence: 42,
					},
				},
				Subscription: config.SubscriptionConfig{Topic: "telegram.serial"},
				Publisher:    config.PublisherConfig{Topic: "telegram.json"},
				DLQ:          config.DLQConfig{Enabled: true, Subject: "caatsm.dlq"},
			}
			handler := MessageHandlerFunc(func(context.Context, []byte, string) error { return nil })
			c, err := NewConsumer(nil, js, handler, cfg, &spyRecorder{}, zap.NewNop())
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
			Expect(js.addedStr).To(Equal([]string{"TELEGRAM", "TELEGRAM_OUT", "TELEGRAM_DLQ"}))
			Expect(js.streams["TELEGRAM"].Config.Retention).To(Equal(nats.WorkQueuePolicy))
			Expect(js.streams["TELEGRAM_OUT"].Config.Retention).To(Equal(nats.LimitsPolicy))
			Expect(js.streams["TELEGRAM_DLQ"].Config.Retention).To(Equal(nats.LimitsPolicy))
			Expect(js.addedCons).To(Equal([]string{"TELEGRAM/c1"}))
			Expect(js.consumers["TELEGRAM/c1"].Config.OptStartSeq).To(Equal(uint64(42)))
		})

		It("does not auto-create resources when auto_provision is false", func() {
			js := &stubJSInfra{
				streams:   map[string]*nats.StreamInfo{},
				consumers: map[string]*nats.ConsumerInfo{},
			}
			auto := false
			cfg := &config.Config{
				NATS: config.NATSConfig{
					Stream:        "TELEGRAM",
					Consumer:      "c1",
					AutoProvision: &auto,
				},
				Subscription: config.SubscriptionConfig{Topic: "telegram.serial"},
				Publisher:    config.PublisherConfig{Topic: "telegram.json"},
			}
			handler := MessageHandlerFunc(func(context.Context, []byte, string) error { return nil })
			_, err := NewConsumer(nil, js, handler, cfg, &spyRecorder{}, zap.NewNop())
			Expect(err).To(HaveOccurred())
			Expect(js.addedStr).To(BeEmpty())
			Expect(js.addedCons).To(BeEmpty())
		})
	})

	Describe("handleFailure", func() {
		It("routes permanent errors to DLQ and ACKs on success", func() {
			rec := &spyRecorder{}
			dlq := &fakeDLQ{}
			c := &Consumer{
				streamName:   "TELEGRAM",
				consumerName: "c1",
				dlqHandler:   dlq,
				recorder:     rec,
				logger:       zap.NewNop(),
			}
			acker := &fakeAckMsg{}
			res := c.handleFailure(context.Background(), context.Background(), &nats.Msg{}, acker, "id-1", markPermanent(errors.New("bad")))

			Expect(res).To(Equal(resultPermanentFail))
			Expect(dlq.called).To(BeTrue())
			Expect(acker.acked).To(BeTrue())
			Expect(acker.nacked).To(BeFalse())
		})

		It("NAKs without ACKing if DLQ routing fails", func() {
			rec := &spyRecorder{}
			dlq := &fakeDLQ{err: errors.New("dlq down")}
			c := &Consumer{
				streamName:   "TELEGRAM",
				consumerName: "c1",
				dlqHandler:   dlq,
				recorder:     rec,
				logger:       zap.NewNop(),
				backoff:      []time.Duration{2 * time.Second},
			}
			acker := &fakeAckMsg{delivered: 1}
			res := c.handleFailure(context.Background(), context.Background(), &nats.Msg{}, acker, "id-2", markPermanent(errors.New("bad")))

			Expect(res).To(Equal(resultFail))
			Expect(acker.acked).To(BeFalse())
			Expect(acker.nacked).To(BeTrue())
			Expect(acker.nakDelay).To(Equal(2 * time.Second))
			Expect(rec.retries).To(Equal([]string{retryReasonDLQError}))
		})

		It("NAKs transient errors with delivery backoff and records retry before maxDeliver", func() {
			rec := &spyRecorder{}
			dlq := &fakeDLQ{}
			c := &Consumer{
				streamName:   "TELEGRAM",
				consumerName: "c1",
				dlqHandler:   dlq,
				recorder:     rec,
				logger:       zap.NewNop(),
				maxDeliver:   5,
				backoff:      []time.Duration{time.Second, 5 * time.Second},
			}
			acker := &fakeAckMsg{delivered: 2}
			res := c.handleFailure(context.Background(), context.Background(), &nats.Msg{}, acker, "id-3", errors.New("db down"))

			Expect(res).To(Equal(resultFail))
			Expect(dlq.called).To(BeFalse())
			Expect(acker.nacked).To(BeTrue())
			Expect(acker.nakDelay).To(Equal(5 * time.Second))
			Expect(rec.retries).To(Equal([]string{retryReasonProcessorError}))
		})

		It("routes transient errors to DLQ and ACKs when maxDeliver is exhausted", func() {
			rec := &spyRecorder{}
			dlq := &fakeDLQ{}
			c := &Consumer{
				streamName:   "TELEGRAM",
				consumerName: "c1",
				dlqHandler:   dlq,
				recorder:     rec,
				logger:       zap.NewNop(),
				maxDeliver:   5,
				backoff:      []time.Duration{time.Second},
			}
			acker := &fakeAckMsg{delivered: 5}
			res := c.handleFailure(context.Background(), context.Background(), &nats.Msg{}, acker, "id-4", errors.New("db persistently down"))

			Expect(res).To(Equal(resultPermanentFail))
			Expect(dlq.called).To(BeTrue())
			Expect(acker.acked).To(BeTrue())
			Expect(acker.nacked).To(BeFalse())
		})

		It("NAKs without ACKing when maxDeliver is exhausted but DLQ publish fails", func() {
			rec := &spyRecorder{}
			dlq := &fakeDLQ{err: errors.New("dlq unavailable")}
			c := &Consumer{
				streamName:   "TELEGRAM",
				consumerName: "c1",
				dlqHandler:   dlq,
				recorder:     rec,
				logger:       zap.NewNop(),
				maxDeliver:   5,
				backoff:      []time.Duration{time.Second},
			}
			acker := &fakeAckMsg{delivered: 5, streamSeq: 42}
			res := c.handleFailure(context.Background(), context.Background(), &nats.Msg{}, acker, "id-5", errors.New("db down"))

			Expect(res).To(Equal(resultFail))
			Expect(dlq.called).To(BeTrue())
			Expect(acker.acked).To(BeFalse())
			Expect(acker.nacked).To(BeTrue())
			Expect(rec.retries).To(Equal([]string{retryReasonDLQError}))
		})

		It("gives DLQ publishing an independent non-expired context when business handling timed out", func() {
			rec := &spyRecorder{}
			dlq := &fakeDLQ{}
			c := &Consumer{
				streamName:   "TELEGRAM",
				consumerName: "c1",
				dlqHandler:   dlq,
				recorder:     rec,
				logger:       zap.NewNop(),
				maxDeliver:   3,
			}
			expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			Expect(expiredCtx.Err()).To(Equal(context.DeadlineExceeded))

			acker := &fakeAckMsg{delivered: 3, streamSeq: 77}
			res := c.handleFailure(context.Background(), expiredCtx, &nats.Msg{Data: []byte("raw")}, acker, "id-timeout", context.DeadlineExceeded)

			Expect(res).To(Equal(resultPermanentFail))
			Expect(dlq.called).To(BeTrue())
			Expect(dlq.lastCtxErr).To(BeNil())
			Expect(dlq.hasDeadln).To(BeTrue())
			Expect(acker.acked).To(BeTrue())
		})

		It("uses broker consumer MaxDeliver over local config when syncing durable metadata", func() {
			js := &stubJSInfra{
				streams: map[string]*nats.StreamInfo{
					"TELEGRAM":     {Config: nats.StreamConfig{Name: "TELEGRAM", Subjects: []string{"telegram.serial"}}},
					"TELEGRAM_OUT": {Config: nats.StreamConfig{Name: "TELEGRAM_OUT", Subjects: []string{"telegram.json"}}},
				},
				consumers: map[string]*nats.ConsumerInfo{
					"TELEGRAM/c1": {
						Stream: "TELEGRAM",
						Name:   "c1",
						Config: nats.ConsumerConfig{Durable: "c1", MaxDeliver: 3},
					},
				},
			}
			auto := false
			cfg := &config.Config{
				NATS: config.NATSConfig{
					Stream:        "TELEGRAM",
					Consumer:      "c1",
					AutoProvision: &auto,
					ConsumerRules: config.ConsumerRulesConfig{MaxDeliver: 5},
				},
				Subscription: config.SubscriptionConfig{Topic: "telegram.serial"},
				Publisher:    config.PublisherConfig{Topic: "telegram.json"},
			}
			c, err := NewConsumer(nil, js, MessageHandlerFunc(func(context.Context, []byte, string) error { return nil }), cfg, &spyRecorder{}, zap.NewNop())
			Expect(err).NotTo(HaveOccurred())
			Expect(c.maxDeliver).To(Equal(3))

			// Broker limit 3 reached on attempt 3 even though local config is 5.
			dlq := &fakeDLQ{}
			c.dlqHandler = dlq
			acker := &fakeAckMsg{delivered: 3, streamSeq: 10}
			Expect(c.handleFailure(context.Background(), context.Background(), &nats.Msg{}, acker, "id-b3", errors.New("tmp"))).To(Equal(resultPermanentFail))
			Expect(dlq.called).To(BeTrue())
			Expect(acker.acked).To(BeTrue())

			// Reverse case: broker limit 5, local config 3 -> attempt 3 keeps retrying.
			c.syncMaxDeliver(&nats.ConsumerInfo{Config: nats.ConsumerConfig{MaxDeliver: 5}})
			Expect(c.maxDeliver).To(Equal(5))
			dlq2 := &fakeDLQ{}
			c.dlqHandler = dlq2
			acker2 := &fakeAckMsg{delivered: 3, streamSeq: 11}
			Expect(c.handleFailure(context.Background(), context.Background(), &nats.Msg{}, acker2, "id-b5", errors.New("tmp"))).To(Equal(resultFail))
			Expect(dlq2.called).To(BeFalse())
			Expect(acker2.nacked).To(BeTrue())
		})
	})

	Describe("Inbound PublishID Derivation", func() {
		It("derives stable unique PublishIDs from JetStream record identity without colliding on upstream Nats-Msg-Id", func() {
			t1 := time.Unix(0, 1700000000000000000).UTC()
			t2 := time.Unix(0, 1700000001000000000).UTC()

			m1 := &nats.MsgMetadata{Stream: "TELEGRAM", Sequence: nats.SequencePair{Stream: 10}, Timestamp: t1}
			m1Redeliver := &nats.MsgMetadata{Stream: "TELEGRAM", Sequence: nats.SequencePair{Stream: 10}, Timestamp: t1, NumDelivered: 2}
			m2 := &nats.MsgMetadata{Stream: "TELEGRAM", Sequence: nats.SequencePair{Stream: 11}, Timestamp: t1}
			mRecreated := &nats.MsgMetadata{Stream: "TELEGRAM", Sequence: nats.SequencePair{Stream: 10}, Timestamp: t2}
			mWithDomain := &nats.MsgMetadata{Domain: "prod", Stream: "TELEGRAM", Sequence: nats.SequencePair{Stream: 10}, Timestamp: t1}

			id1 := formatInboundPublishID(m1, "TELEGRAM")
			Expect(formatInboundPublishID(m1Redeliver, "TELEGRAM")).To(Equal(id1))
			Expect(formatInboundPublishID(m2, "TELEGRAM")).NotTo(Equal(id1))
			Expect(formatInboundPublishID(mRecreated, "TELEGRAM")).NotTo(Equal(id1))
			Expect(formatInboundPublishID(mWithDomain, "TELEGRAM")).To(ContainSubstring("prod"))

			// Two distinct JetStream messages sharing the same upstream Nats-Msg-Id receive distinct PublishIDs.
			var seenPubIDs []string
			var seenCorrIDs []string
			c := &Consumer{
				streamName:   "TELEGRAM",
				consumerName: "c1",
				recorder:     &spyRecorder{},
				logger:       zap.NewNop(),
				monitor:      NewConsumerMonitor(zap.NewNop(), nil, nil, "TELEGRAM", "c1", time.Second, nil),
				handler: MessageHandlerFunc(func(ctx context.Context, _ []byte, msgID string) error {
					seenPubIDs = append(seenPubIDs, telegram.PublishIDFromContext(ctx))
					seenCorrIDs = append(seenCorrIDs, msgID)
					return nil
				}),
			}
			msgA := &nats.Msg{
				Data:   []byte("payload-a"),
				Reply:  "$JS.ACK.TELEGRAM.c1.1.10.10.1700000000000000000.0",
				Sub:    &nats.Subscription{},
				Header: nats.Header{"Nats-Msg-Id": []string{"shared-upstream-id"}},
			}
			msgB := &nats.Msg{
				Data:   []byte("payload-b"),
				Reply:  "$JS.ACK.TELEGRAM.c1.1.11.11.1700000000000000000.0",
				Sub:    &nats.Subscription{},
				Header: nats.Header{"Nats-Msg-Id": []string{"shared-upstream-id"}},
			}
			c.processMsg(context.Background(), msgA)
			c.processMsg(context.Background(), msgB)

			Expect(seenCorrIDs).To(Equal([]string{"shared-upstream-id", "shared-upstream-id"}))
			Expect(seenPubIDs).To(HaveLen(2))
			Expect(seenPubIDs[0]).NotTo(BeEmpty())
			Expect(seenPubIDs[0]).NotTo(Equal(seenPubIDs[1]))
		})

		It("retries instead of falling back to upstream ID when JetStream metadata cannot be parsed", func() {
			rec := &spyRecorder{}
			handlerCalled := false
			c := &Consumer{
				streamName:   "TELEGRAM",
				consumerName: "c1",
				recorder:     rec,
				logger:       zap.NewNop(),
				monitor:      NewConsumerMonitor(zap.NewNop(), nil, nil, "TELEGRAM", "c1", time.Second, nil),
				handler: MessageHandlerFunc(func(context.Context, []byte, string) error {
					handlerCalled = true
					return nil
				}),
			}
			badMsg := &nats.Msg{
				Data:   []byte("payload"),
				Header: nats.Header{"Nats-Msg-Id": []string{"upstream-1"}},
			}
			c.processMsg(context.Background(), badMsg)
			Expect(handlerCalled).To(BeFalse())
			Expect(rec.retries).To(Equal([]string{retryReasonProcessorError}))
		})
	})

	Describe("Backoff, Monitor, and Batch Cancellation", func() {
		It("caps exponential fetch backoff at 5s", func() {
			Expect(fetchBackoff(1)).To(Equal(500 * time.Millisecond))
			Expect(fetchBackoff(2)).To(Equal(1 * time.Second))
			Expect(fetchBackoff(3)).To(Equal(2 * time.Second))
			Expect(fetchBackoff(4)).To(Equal(4 * time.Second))
			Expect(fetchBackoff(5)).To(Equal(5 * time.Second))
			Expect(fetchBackoff(10)).To(Equal(5 * time.Second))
		})

		It("ignores redeliveries and out-of-order replays in sequence gap detection", func() {
			rec := &spyRecorder{}
			cfg := &config.Config{
				AFTN: config.AFTNConfig{EnableSequenceGapDetection: true},
			}
			mon := NewConsumerMonitor(zap.NewNop(), cfg, nil, "TELEGRAM", "c1", time.Second, rec)
			now := time.Now()

			mon.recordSequenceObservation(10, 1, now)
			// Redelivery of an older sequence must not lower high-water mark or trigger a gap.
			mon.recordSequenceObservation(5, 2, now)
			// Next fresh delivery in sequence: 11 -> no gap.
			mon.recordSequenceObservation(11, 1, now)
			Expect(rec.seqGaps).To(BeEmpty())

			// True gap: 11 -> 15 (3 missing).
			mon.recordSequenceObservation(15, 1, now)
			Expect(rec.seqGaps).To(Equal([]uint64{3}))
		})

		It("stops processing remaining batch messages when context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			handled := 0
			c := &Consumer{
				streamName:   "TELEGRAM",
				consumerName: "c1",
				recorder:     &spyRecorder{},
				logger:       zap.NewNop(),
				monitor:      NewConsumerMonitor(zap.NewNop(), nil, nil, "TELEGRAM", "c1", time.Second, nil),
				handler: MessageHandlerFunc(func(_ context.Context, _ []byte, _ string) error {
					handled++
					cancel()
					return nil
				}),
			}

			msgs := []*nats.Msg{
				{Data: []byte("m1"), Reply: "$JS.ACK.TELEGRAM.c1.1.1.1.1700000000000000000.2", Sub: &nats.Subscription{}},
				{Data: []byte("m2"), Reply: "$JS.ACK.TELEGRAM.c1.1.2.2.1700000000000000000.1", Sub: &nats.Subscription{}},
				{Data: []byte("m3"), Reply: "$JS.ACK.TELEGRAM.c1.1.3.3.1700000000000000000.0", Sub: &nats.Subscription{}},
			}
			c.processBatch(ctx, msgs)
			Expect(handled).To(Equal(1))
		})

		It("stops ConsumerMonitor cleanly on context cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			js := &stubJSInfra{streams: map[string]*nats.StreamInfo{}, consumers: map[string]*nats.ConsumerInfo{}}
			rec := &spyRecorder{}
			mon := NewConsumerMonitor(zap.NewNop(), nil, js, "TELEGRAM", "c1", 10*time.Millisecond, rec)
			mon.emitStats(ctx)
			Expect(rec.jsCalls).To(Equal(1))
		})
	})
})

type MessageHandlerFunc func(context.Context, []byte, string) error

func (f MessageHandlerFunc) Handle(ctx context.Context, raw []byte, msgID string) error {
	return f(ctx, raw, msgID)
}
