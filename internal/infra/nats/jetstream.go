package nats

import (
	"caatsm/internal/infra/config"
	"fmt"
	"go.uber.org/zap"
	"github.com/nats-io/nats.go"
	"time"
)

// ProvideJetStream creates a NATS JetStream connection
func ProvideJetStream(cfg *config.Config, logger *zap.Logger) (nats.JetStreamContext, error) {
	// Connect to NATS
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

	streamConfig := &nats.StreamConfig{
		Name:      streamName,
		Subjects:  []string{subject},
		Retention: nats.LimitsPolicy,
		MaxAge:    24 * time.Hour,
		Storage:   nats.FileStorage,
		Replicas:  1,
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
