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
	subject := cfg.Subscription.Topic
	if subject == "" {
		subject = "telegram.>"
	}

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

	_, err = js.AddStream(streamConfig)
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		nc.Close()
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	if err == nil {
		logger.Info("Created JetStream", zap.String("stream", streamName), zap.String("subject", subject))
	}

	return js, nil
}
