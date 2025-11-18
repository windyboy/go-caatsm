package nats

import (
	"caatsm/internal/infra/config"
	"errors"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// StreamManager handles JetStream stream lifecycle management
type StreamManager struct {
	js         nats.JetStreamContext
	streamName string
	subjects   []string
	logger     *zap.Logger
	cfg        *config.StreamLimitsConfig
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

// NewStreamManagerWithConfig creates a new stream manager with full stream configuration
func NewStreamManagerWithConfig(js nats.JetStreamContext, streamName string, subjects []string, streamLimits *config.StreamLimitsConfig, logger *zap.Logger) *StreamManager {
	return &StreamManager{
		js:         js,
		streamName: streamName,
		subjects:   subjects,
		logger:     logger,
		cfg:        streamLimits,
	}
}

// EnsureStream ensures that the configured JetStream stream exists
func (sm *StreamManager) EnsureStream() error {
	// Build stream configuration
	streamConfig := sm.buildStreamConfig()

	info, err := sm.js.StreamInfo(sm.streamName)
	if err != nil {
		if errors.Is(err, nats.ErrStreamNotFound) {
			if shouldBootstrapStream() {
				if _, err = sm.js.AddStream(streamConfig); err != nil {
					sm.logger.Error("failed to create stream",
						zap.String("stream", sm.streamName),
						zap.Strings("subjects", sm.subjects),
						zap.Error(err),
					)
					return fmt.Errorf("failed to create stream %s: %w", sm.streamName, err)
				}
				sm.logger.Info("Created JetStream stream",
					zap.String("stream", sm.streamName),
					zap.Strings("subjects", sm.subjects),
				)
				return nil
			}
			sm.logger.Error("stream not found and auto-creation disabled",
				zap.String("stream", sm.streamName),
				zap.Strings("expected_subjects", sm.subjects),
			)
			return fmt.Errorf("stream %s not found and auto-creation disabled", sm.streamName)
		}
		sm.logger.Error("failed to fetch stream info",
			zap.String("stream", sm.streamName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to fetch stream info for %s: %w", sm.streamName, err)
	}

	// Stream exists: validate subjects but do not fail hard if they differ.
	sm.validateStreamConfig(info)
	return nil
}

// buildStreamConfig builds the stream configuration from manager settings
func (sm *StreamManager) buildStreamConfig() *nats.StreamConfig {
	config := &nats.StreamConfig{
		Name:      sm.streamName,
		Subjects:  sm.subjects,
		Retention: nats.LimitsPolicy,
		Storage:   nats.FileStorage,
	}

	// Apply stream limits configuration if provided
	if sm.cfg != nil {
		config.MaxMsgs = sm.cfg.MaxMsgs
		config.MaxBytes = sm.cfg.MaxBytes
		config.MaxAge = sm.cfg.MaxAge
		config.Replicas = sm.cfg.Replicas

		// Map storage type
		switch strings.ToLower(sm.cfg.Storage) {
		case "memory":
			config.Storage = nats.MemoryStorage
		case "file":
			config.Storage = nats.FileStorage
		}

		// Map discard policy
		if strings.EqualFold(sm.cfg.Discard, "new") {
			config.Discard = nats.DiscardNew
		} else {
			config.Discard = nats.DiscardOld
		}
	}

	return config
}

// validateStreamConfig validates the stream configuration
func (sm *StreamManager) validateStreamConfig(info *nats.StreamInfo) {
	if info == nil {
		return
	}

	missing := make([]string, 0)
	for _, subj := range sm.subjects {
		if subj == "" {
			continue
		}
		if !containsSubject(info.Config.Subjects, subj) {
			missing = append(missing, subj)
		}
	}
	if len(missing) > 0 {
		sm.logger.Warn("JetStream stream subjects missing expected entries",
			zap.String("stream", info.Config.Name),
			zap.Strings("stream_subjects", info.Config.Subjects),
			zap.Strings("missing_subjects", missing),
		)
	}
}
