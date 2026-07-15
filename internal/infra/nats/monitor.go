package nats

import (
	"caatsm/internal/infra/config"
	obsmetrics "caatsm/internal/infra/metrics"
	"context"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ConsumerMonitor handles health monitoring and stats emission for the consumer.
type ConsumerMonitor struct {
	logger *zap.Logger
	cfg    *config.Config
	js     nats.JetStreamContext

	// Configuration
	streamName      string
	consumerName    string
	monitorInterval time.Duration

	// State
	lastMessageTime     time.Time
	lastMessageSequence uint64
	messageGapMutex     sync.RWMutex
}

// NewConsumerMonitor creates a new ConsumerMonitor.
func NewConsumerMonitor(
	logger *zap.Logger,
	cfg *config.Config,
	js nats.JetStreamContext,
	streamName string,
	consumerName string,
	monitorInterval time.Duration,
) *ConsumerMonitor {
	return &ConsumerMonitor{
		logger:          logger,
		cfg:             cfg,
		js:              js,
		streamName:      streamName,
		consumerName:    consumerName,
		monitorInterval: monitorInterval,
	}
}

// Start begins the monitoring loop.
func (m *ConsumerMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.monitorInterval)
	defer ticker.Stop()

	// Record initial metric (0) to ensure it appears in Prometheus even before first tick
	m.logger.Info("Starting NATS consumer stats emission goroutine, recording initial pending metric",
		zap.String("stream", m.streamName),
		zap.String("consumer", m.consumerName),
		zap.Duration("interval", m.monitorInterval),
	)
	obsmetrics.RecordNATSConsumerPending(m.streamName, m.consumerName, 0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.emitStats()
		}
	}
}

// emitStats gathers and records consumer statistics.
func (m *ConsumerMonitor) emitStats() {
	if m.js != nil {
		info, err := m.js.ConsumerInfo(m.streamName, m.consumerName)
		if err != nil {
			m.logger.Warn("Failed to fetch consumer info for pending messages metric",
				zap.String("stream", m.streamName),
				zap.String("consumer", m.consumerName),
				zap.Error(err),
			)
		} else {
			// Record pending messages for monitoring
			m.logger.Debug("Recording NATS consumer pending messages metric",
				zap.String("stream", m.streamName),
				zap.String("consumer", m.consumerName),
				zap.Uint64("pending", info.NumPending),
			)
			obsmetrics.RecordNATSConsumerPending(m.streamName, m.consumerName, info.NumPending)
		}
	}

	// Record AFTN health metrics
	gapSeconds := m.GetMessageGapSeconds()
	healthy := m.IsHealthy()

	obsmetrics.RecordMessageGap(m.streamName, m.consumerName, gapSeconds)
	obsmetrics.RecordSerialReaderHealth(m.streamName, m.consumerName, healthy)

	if !healthy {
		m.logger.Warn("Serial reader appears stalled - no messages received recently",
			zap.String("stream", m.streamName),
			zap.String("consumer", m.consumerName),
			zap.Float64("gap_seconds", gapSeconds),
			zap.Duration("threshold", m.cfg.AFTN.MessageGapThreshold),
		)
	}
}

// TrackMessage updates health metrics based on a received message.
func (m *ConsumerMonitor) TrackMessage(msg *nats.Msg) {
	if msg == nil {
		return
	}

	m.messageGapMutex.Lock()
	defer m.messageGapMutex.Unlock()

	now := time.Now()
	m.lastMessageTime = now

	// Extract sequence number from message metadata
	if meta, err := msg.Metadata(); err == nil {
		currentSeq := meta.Sequence.Stream

		// Detect sequence gaps if we have a previous sequence
		if m.lastMessageSequence > 0 && m.cfg.AFTN.EnableSequenceGapDetection {
			if currentSeq > m.lastMessageSequence+1 {
				gapSize := currentSeq - m.lastMessageSequence - 1
				m.logger.Warn("Message sequence gap detected",
					zap.String("stream", m.streamName),
					zap.String("consumer", m.consumerName),
					zap.Uint64("last_sequence", m.lastMessageSequence),
					zap.Uint64("current_sequence", currentSeq),
					zap.Uint64("gap_size", gapSize),
				)
				obsmetrics.RecordSequenceGap(m.streamName, m.consumerName, gapSize)
			}
		}

		m.lastMessageSequence = currentSeq
	}
}

// GetMessageGapSeconds returns the number of seconds since the last message was received.
func (m *ConsumerMonitor) GetMessageGapSeconds() float64 {
	m.messageGapMutex.RLock()
	defer m.messageGapMutex.RUnlock()

	if m.lastMessageTime.IsZero() {
		return 0
	}

	return time.Since(m.lastMessageTime).Seconds()
}

// IsHealthy returns true if messages are being received within the threshold.
func (m *ConsumerMonitor) IsHealthy() bool {
	m.messageGapMutex.RLock()
	defer m.messageGapMutex.RUnlock()

	// If we haven't received any messages yet, consider it healthy (initial state)
	if m.lastMessageTime.IsZero() {
		return true
	}

	gap := time.Since(m.lastMessageTime)
	return gap < m.cfg.AFTN.MessageGapThreshold
}

// RecordInitialPending logs and records the initial pending messages count.
// This is exposed to allow recording immediately upon startup.
func (m *ConsumerMonitor) RecordInitialPending(pending uint64) {
	m.logger.Info("Recording initial NATS consumer pending messages metric",
		zap.String("stream", m.streamName),
		zap.String("consumer", m.consumerName),
		zap.Uint64("pending", pending),
	)
	obsmetrics.RecordNATSConsumerPending(m.streamName, m.consumerName, pending)
}
