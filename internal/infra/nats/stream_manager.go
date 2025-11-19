package nats

import (
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// StreamManager handles JetStream stream lifecycle management
type StreamManager struct {
	js         nats.JetStreamContext
	streamName string
	subjects   []string
	logger     *zap.Logger
}

// StreamConfig holds configuration for creating a JetStream stream
type StreamConfig struct {
	MaxMsgs   int64
	MaxBytes  int64
	MaxAge    time.Duration
	Discard   string // "old" or "new"
	Storage   string // "file" or "memory"
	Replicas  int
}

// NewStreamManager creates a new stream manager
func NewStreamManager(js nats.JetStreamContext, streamName string, subjects []string, logger *zap.Logger) *StreamManager {
	return &StreamManager{
		js:         js,
		streamName: streamName,
		subjects:   subjects,
		logger:     logger,
	}
}

// EnsureStream ensures that the configured JetStream stream exists, creating it if necessary
func (sm *StreamManager) EnsureStream(cfg *StreamConfig) error {
	// Check if stream already exists
	_, err := sm.js.StreamInfo(sm.streamName)
	if err == nil {
		sm.logger.Info("JetStream stream verified",
			zap.String("stream", sm.streamName),
			zap.Strings("subjects", sm.subjects),
		)
		return nil
	}

	// If stream doesn't exist, create it
	if errors.Is(err, nats.ErrStreamNotFound) {
		streamCfg := &nats.StreamConfig{
			Name:     sm.streamName,
			Subjects: sm.subjects,
		}

		// Apply limits if provided
		if cfg != nil {
			if cfg.MaxMsgs > 0 {
				streamCfg.MaxMsgs = cfg.MaxMsgs
			}
			if cfg.MaxBytes > 0 {
				streamCfg.MaxBytes = cfg.MaxBytes
			}
			if cfg.MaxAge > 0 {
				streamCfg.MaxAge = cfg.MaxAge
			}
			if cfg.Discard == "new" {
				streamCfg.Discard = nats.DiscardNew
			} else {
				streamCfg.Discard = nats.DiscardOld
			}
			if cfg.Storage == "memory" {
				streamCfg.Storage = nats.MemoryStorage
			} else {
				streamCfg.Storage = nats.FileStorage
			}
			if cfg.Replicas > 0 {
				streamCfg.Replicas = cfg.Replicas
			}
		}

		_, err := sm.js.AddStream(streamCfg)
		if err != nil {
			return fmt.Errorf("failed to create stream %s: %w", sm.streamName, err)
		}

		sm.logger.Info("Created JetStream stream",
			zap.String("stream", sm.streamName),
			zap.Strings("subjects", sm.subjects),
		)
		return nil
	}

	// Other error (e.g., permission denied)
	return fmt.Errorf("stream %s not found or inaccessible: %w", sm.streamName, err)
}
