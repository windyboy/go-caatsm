package nats

import (
	"fmt"

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

// NewStreamManager creates a new stream manager
func NewStreamManager(js nats.JetStreamContext, streamName string, subjects []string, logger *zap.Logger) *StreamManager {
	return &StreamManager{
		js:         js,
		streamName: streamName,
		subjects:   subjects,
		logger:     logger,
	}
}

// EnsureStream ensures that the configured JetStream stream exists
func (sm *StreamManager) EnsureStream() error {
	_, err := sm.js.StreamInfo(sm.streamName)
	if err != nil {
		return fmt.Errorf("stream %s not found or inaccessible: %w", sm.streamName, err)
	}

	sm.logger.Info("JetStream stream verified",
		zap.String("stream", sm.streamName),
		zap.Strings("subjects", sm.subjects),
	)
	return nil
}
