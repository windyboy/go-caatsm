package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/observability"
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ConsumerMonitor handles health monitoring and backlog metrics for the consumer.
type ConsumerMonitor struct {
	logger *zap.Logger
	cfg    *config.Config
	js     nats.JetStreamContext

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
) *ConsumerMonitor {
	if logger == nil {
		logger = zap.NewNop()
	}
	if monitorInterval <= 0 {
		monitorInterval = 30 * time.Second
	}
	return &ConsumerMonitor{
		logger:          logger,
		cfg:             cfg,
		js:              js,
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
	observability.RecordNATSConsumerPending(m.streamName, m.consumerName, 0)

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
		observability.RecordJSAPICall("consumer_info")
		info, err := m.js.ConsumerInfo(m.streamName, m.consumerName)
		if err != nil {
			m.logger.Warn("Failed to fetch consumer info for pending messages metric",
				zap.String("stream", m.streamName),
				zap.String("consumer", m.consumerName),
				zap.Error(err),
			)
		} else {
			observability.RecordNATSConsumerPending(m.streamName, m.consumerName, info.NumPending)
		}
	}

	gapSeconds := m.GetMessageGapSeconds()
	healthy := m.IsHealthy()

	observability.RecordMessageGap(m.streamName, m.consumerName, gapSeconds)
	observability.RecordSerialReaderHealth(m.streamName, m.consumerName, healthy)

	if !healthy {
		m.logger.Warn("Serial reader appears stalled - no messages received recently",
			zap.String("stream", m.streamName),
			zap.String("consumer", m.consumerName),
			zap.Float64("gap_seconds", gapSeconds),
			zap.Duration("threshold", m.messageGapThreshold()),
		)
	}
}

// TrackMessage updates health and sequence-gap metrics on each received message.
func (m *ConsumerMonitor) TrackMessage(msg *nats.Msg) {
	if msg == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastMessageTime = time.Now()

	if meta, err := msg.Metadata(); err == nil {
		currentSeq := meta.Sequence.Stream
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
				observability.RecordSequenceGap(m.streamName, m.consumerName, gapSize)
			}
		}
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
	observability.RecordNATSConsumerPending(m.streamName, m.consumerName, pending)
}

func (m *ConsumerMonitor) messageGapThreshold() time.Duration {
	if m.cfg != nil && m.cfg.AFTN.MessageGapThreshold > 0 {
		return m.cfg.AFTN.MessageGapThreshold
	}
	return 2 * time.Minute
}
