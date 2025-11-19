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
	info, err := sm.js.StreamInfo(sm.streamName)
	if err == nil {
		// Stream exists - check if we need to add any missing subjects
		existingSubjects := make(map[string]bool)
		for _, subj := range info.Config.Subjects {
			existingSubjects[subj] = true
		}
		
		// Check if any configured subjects are missing
		missingSubjects := []string{}
		for _, subj := range sm.subjects {
			if !existingSubjects[subj] {
				missingSubjects = append(missingSubjects, subj)
			}
		}
		
		if len(missingSubjects) > 0 {
			// Update stream to include missing subjects
			updatedSubjects := info.Config.Subjects
			updatedSubjects = append(updatedSubjects, missingSubjects...)
			info.Config.Subjects = updatedSubjects
			
			_, updateErr := sm.js.UpdateStream(&info.Config)
			if updateErr != nil {
				return fmt.Errorf("failed to update stream %s with new subjects %v: %w", sm.streamName, missingSubjects, updateErr)
			}
			
			sm.logger.Info("Updated JetStream stream with new subjects",
				zap.String("stream", sm.streamName),
				zap.Strings("added_subjects", missingSubjects),
				zap.Strings("all_subjects", updatedSubjects),
			)
		} else {
			sm.logger.Info("JetStream stream verified",
				zap.String("stream", sm.streamName),
				zap.Strings("subjects", sm.subjects),
			)
		}
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
