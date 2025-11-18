package nats

import (
	"caatsm/internal/infra/config"
	"fmt"
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
		// Use infinite reconnects so the app survives long NATS outages (e.g. docker compose down/up).
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			safeURL := sanitizeURLForLogging(nc.ConnectedUrl())
			logger.Info("NATS reconnected", zap.String("url", safeURL))
		}),
	)
	if err != nil {
		safeURL := sanitizeURLForLogging(cfg.NATS.URL)
		logger.Error("failed to connect to NATS",
			zap.String("url", safeURL),
			zap.Duration("timeout", cfg.Timeouts.Server),
			zap.Duration("reconnect_wait", cfg.Timeouts.ReconnectWait),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return nc, nil
}

// ProvideJetStream creates a NATS JetStream context using an existing connection.
// Returns nil, nil when cfg.NATS.Mode == "core" to support plain NATS servers without JetStream.
func ProvideJetStream(nc *nats.Conn, cfg *config.Config, logger *zap.Logger) (nats.JetStreamContext, error) {
	mode := strings.ToLower(cfg.NATS.Mode)
	if mode == "core" {
		logger.Info("Skipping JetStream initialization for core NATS mode")
		return nil, nil
	}

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		safeURL := sanitizeURLForLogging(cfg.NATS.URL)
		logger.Error("failed to get JetStream context",
			zap.String("url", safeURL),
			zap.Error(err),
		)
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Ensure the stream exists using StreamManager
	streamName := cfg.NATS.Stream
	consumerSubject := cfg.EffectiveSubscriptionTopic()
	publisherSubject := strings.TrimSpace(cfg.Publisher.Topic)

	streamSubjects := dedupeSubjects([]string{consumerSubject, publisherSubject})
	if len(streamSubjects) == 0 {
		logger.Error("no subjects configured for JetStream stream",
			zap.String("stream", streamName),
			zap.String("consumer_subject", consumerSubject),
			zap.String("publisher_subject", publisherSubject),
		)
		nc.Close()
		return nil, fmt.Errorf("no subjects configured for JetStream stream %s", streamName)
	}

	streamManager := NewStreamManagerWithConfig(js, streamName, streamSubjects, &cfg.NATS.StreamLimits, logger)
	if err := streamManager.EnsureStream(); err != nil {
		nc.Close()
		return nil, err
	}

	return js, nil
}
