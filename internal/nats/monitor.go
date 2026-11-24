package nats

import (
	"caatsm/internal/config"
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// MonitorRecorder captures NATS consumer monitoring metrics.
type MonitorRecorder interface {
	RecordJSAPICall(operation string)
	RecordNATSConsumerPending(stream, consumer string, pending uint64)
	RecordMessageGap(stream, consumer string, gapSeconds float64)
	RecordSequenceGap(stream, consumer string, gapSize uint64)
	RecordSerialReaderHealth(stream, consumer string, healthy bool)
}

// ConsumerMonitor handles health monitoring and backlog metrics for the consumer.
type ConsumerMonitor struct {
	logger   *zap.Logger
	cfg      *config.Config
	js       nats.JetStreamContext
	recorder MonitorRecorder

	streamName      string
	consumerName    string
	monitorInterval time.Duration

	lastMessageTime     time.Time
	lastMessageSequence uint64
	mu                  sync.RWMutex
}

// NewConsumerMonitor creates a ConsumerMonitor.
func NewConsumerMonitor(
	logger *zap.Logger,
	cfg *config.Config,
	js nats.JetStreamContext,
	streamName string,
	consumerName string,
	monitorInterval time.Duration,
	rec MonitorRecorder,
) *ConsumerMonitor {
	if logger == nil {
		logger = zap.NewNop()
	}
	if monitorInterval <= 0 {
		monitorInterval = 30 * time.Second
	}
	if rec == nil {
		rec = noopRecorder{}
	}
	return &ConsumerMonitor{
		logger:          logger,
		cfg:             cfg,
		js:              js,
		recorder:        rec,
		streamName:      streamName,
		consumerName:    consumerName,
		monitorInterval: monitorInterval,
	}
}

// Start runs the periodic monitoring loop until ctx is cancelled.
func (m *ConsumerMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.monitorInterval)
	defer ticker.Stop()

	m.logger.Info("Starting NATS consumer stats emission goroutine",
		zap.String("stream", m.streamName),
		zap.String("consumer", m.consumerName),
		zap.Duration("interval", m.monitorInterval),
	)
	m.recorder.RecordNATSConsumerPending(m.streamName, m.consumerName, 0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.emitStats()
		}
	}
}

func (m *ConsumerMonitor) emitStats() {
	if m.js != nil {
		m.recorder.RecordJSAPICall("consumer_info")
		info, err := m.js.ConsumerInfo(m.streamName, m.consumerName)
		if err != nil {
			m.logger.Warn("Failed to fetch consumer info for pending messages metric",
				zap.String("stream", m.streamName),
				zap.String("consumer", m.consumerName),
				zap.Error(err),
			)
		} else {
			m.recorder.RecordNATSConsumerPending(m.streamName, m.consumerName, info.NumPending)
		}
	}

	gapSeconds := m.GetMessageGapSeconds()
	healthy := m.IsHealthy()

	m.recorder.RecordMessageGap(m.streamName, m.consumerName, gapSeconds)
	m.recorder.RecordSerialReaderHealth(m.streamName, m.consumerName, healthy)

	if !healthy {
		m.logger.Warn("Serial reader appears stalled - no messages received recently",
			zap.String("stream", m.streamName),
			zap.String("consumer", m.consumerName),
			zap.Float64("gap_seconds", gapSeconds),
			zap.Duration("threshold", m.messageGapThreshold()),
		)
	}
}

// TrackMessage updates health and monotonic sequence-gap metrics on newly delivered messages.
func (m *ConsumerMonitor) TrackMessage(msg *nats.Msg) {
	if msg == nil {
		return
	}

	meta, err := msg.Metadata()
	if err == nil && meta != nil {
		m.recordSequenceObservation(meta.Sequence.Stream, meta.NumDelivered, time.Now())
		return
	}

	m.mu.Lock()
	m.lastMessageTime = time.Now()
	m.mu.Unlock()
}

func (m *ConsumerMonitor) recordSequenceObservation(currentSeq, numDelivered uint64, observedAt time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastMessageTime = observedAt

	// Skip redeliveries and out-of-order replays so they never trigger false sequence gaps
	// or move the monotonic high-water mark backwards.
	if numDelivered > 1 || currentSeq == 0 {
		return
	}
	if m.lastMessageSequence > 0 && m.cfg != nil && m.cfg.AFTN.EnableSequenceGapDetection {
		if currentSeq > m.lastMessageSequence+1 {
			gapSize := currentSeq - m.lastMessageSequence - 1
			m.logger.Warn("Message sequence gap detected",
				zap.String("stream", m.streamName),
				zap.String("consumer", m.consumerName),
				zap.Uint64("last_sequence", m.lastMessageSequence),
				zap.Uint64("current_sequence", currentSeq),
				zap.Uint64("gap_size", gapSize),
			)
			m.recorder.RecordSequenceGap(m.streamName, m.consumerName, gapSize)
		}
	}
	if currentSeq > m.lastMessageSequence {
		m.lastMessageSequence = currentSeq
	}
}

// GetMessageGapSeconds returns seconds elapsed since the last message arrived.
func (m *ConsumerMonitor) GetMessageGapSeconds() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastMessageTime.IsZero() {
		return 0
	}
	return time.Since(m.lastMessageTime).Seconds()
}

// IsHealthy reports whether messages are flowing within the configured threshold.
func (m *ConsumerMonitor) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastMessageTime.IsZero() {
		return true
	}
	return time.Since(m.lastMessageTime) < m.messageGapThreshold()
}

// RecordInitialPending logs and records the startup pending message backlog.
func (m *ConsumerMonitor) RecordInitialPending(pending uint64) {
	m.logger.Info("Recording initial NATS consumer pending messages metric",
		zap.String("stream", m.streamName),
		zap.String("consumer", m.consumerName),
		zap.Uint64("pending", pending),
	)
	m.recorder.RecordNATSConsumerPending(m.streamName, m.consumerName, pending)
}

func (m *ConsumerMonitor) messageGapThreshold() time.Duration {
	if m.cfg != nil && m.cfg.AFTN.MessageGapThreshold > 0 {
		return m.cfg.AFTN.MessageGapThreshold
	}
	return 2 * time.Minute
}
