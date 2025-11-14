package nats

import (
	"caatsm/internal/infra/config"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ProvideNATSConn creates a reusable NATS connection.
func ProvideNATSConn(cfg *config.Config, logger *zap.Logger) (*nats.Conn, error) {
	nc, err := nats.Connect(
		cfg.NATS.URL,
		nats.RetryOnFailedConnect(true),
		nats.Timeout(cfg.Timeouts.Server),
		nats.ReconnectWait(cfg.Timeouts.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return nc, nil
}

// ProvideJetStream creates a NATS JetStream context using an existing connection.
func ProvideJetStream(nc *nats.Conn, cfg *config.Config, logger *zap.Logger) (nats.JetStreamContext, error) {
	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Create stream if it doesn't exist
	streamName := cfg.NATS.Stream
	subject := cfg.EffectiveSubscriptionTopic()

	streamLimits := cfg.NATS.StreamLimits
	storage := nats.FileStorage
	switch strings.ToLower(streamLimits.Storage) {
	case "memory":
		storage = nats.MemoryStorage
	case "file":
		storage = nats.FileStorage
	}

	discard := nats.DiscardOld
	if strings.EqualFold(streamLimits.Discard, "new") {
		discard = nats.DiscardNew
	}

	streamConfig := &nats.StreamConfig{
		Name:      streamName,
		Subjects:  []string{subject},
		Retention: nats.LimitsPolicy,
		MaxMsgs:   streamLimits.MaxMsgs,
		MaxBytes:  streamLimits.MaxBytes,
		MaxAge:    streamLimits.MaxAge,
		Discard:   discard,
		Storage:   storage,
		Replicas:  streamLimits.Replicas,
	}

	info, err := js.StreamInfo(streamName)
	if err != nil {
		if errors.Is(err, nats.ErrStreamNotFound) {
			if shouldBootstrapStream() {
				if _, err = js.AddStream(streamConfig); err != nil {
					nc.Close()
					return nil, fmt.Errorf("failed to create stream: %w", err)
				}
				logger.Info("Created JetStream", zap.String("stream", streamName), zap.String("subject", subject))
			} else {
				nc.Close()
				return nil, fmt.Errorf("stream %s not found and auto-creation disabled", streamName)
			}
		} else {
			nc.Close()
			return nil, fmt.Errorf("failed to fetch stream info: %w", err)
		}
	} else {
		validateStreamConfig(info, subject, logger)
	}

	return js, nil
}

func shouldBootstrapStream() bool {
	switch strings.ToLower(os.Getenv("GO_ENV")) {
	case "", "dev", "development", "test", "testing":
		return true
	default:
		return false
	}
}

func validateStreamConfig(info *nats.StreamInfo, expectedSubject string, logger *zap.Logger) {
	if info == nil {
		return
	}

	if !subjectListContains(info.Config.Subjects, expectedSubject) {
		logger.Warn("JetStream stream subjects do not match config",
			zap.String("stream", info.Config.Name),
			zap.Strings("stream_subjects", info.Config.Subjects),
			zap.String("configured_subject", expectedSubject),
		)
	}
}

func subjectListContains(subjects []string, target string) bool {
	for _, s := range subjects {
		if s == target {
			return true
		}
	}
	return false
}
