package nats

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ConsumerManager handles JetStream consumer lifecycle management
type ConsumerManager struct {
	js           nats.JetStreamContext
	streamName   string
	consumerName string
	subject      string
	logger       *zap.Logger
}

// NewConsumerManager creates a new consumer manager
func NewConsumerManager(js nats.JetStreamContext, streamName, consumerName, subject string, logger *zap.Logger) *ConsumerManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ConsumerManager{
		js:           js,
		streamName:   streamName,
		consumerName: consumerName,
		subject:      subject,
		logger:       logger,
	}
}

// EnsureConsumer creates the consumer if it doesn't exist; if it already exists, it is reused.
func (cm *ConsumerManager) EnsureConsumer(config *nats.ConsumerConfig) error {
	// First check if the consumer already exists to make this initialization idempotent.
	info, err := cm.js.ConsumerInfo(cm.streamName, cm.consumerName)
	if err == nil && info != nil {
		cm.logger.Info("Using existing JetStream consumer",
			zap.String("consumer", cm.consumerName),
			zap.String("stream", cm.streamName),
			zap.String("subject", cm.subject),
		)
		return nil
	}
	if err != nil && !errors.Is(err, nats.ErrConsumerNotFound) {
		return fmt.Errorf("failed to fetch consumer info: %w", err)
	}

	// Consumer does not exist; create it.
	if _, err := cm.js.AddConsumer(cm.streamName, config); err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	cm.logger.Info("Created JetStream consumer",
		zap.String("consumer", cm.consumerName),
		zap.String("stream", cm.streamName),
		zap.String("subject", cm.subject),
		zap.Duration("ack_wait", config.AckWait),
	)

	return nil
}

// CreatePullSubscription creates a pull subscription with recovery logic
func (cm *ConsumerManager) CreatePullSubscription() (*nats.Subscription, error) {
	return cm.js.PullSubscribe(cm.subject, cm.consumerName, nats.Bind(cm.streamName, cm.consumerName))
}

// CreatePullSubscriptionWithRecovery creates a pull subscription
func (cm *ConsumerManager) CreatePullSubscriptionWithRecovery(streamManager *StreamManager, consumerConfig *nats.ConsumerConfig) (*nats.Subscription, error) {
	sub, err := cm.CreatePullSubscription()
	if err == nil {
		return sub, nil
	}

	// Attempt recovery when the consumer or stream is missing.
	if !errors.Is(err, nats.ErrConsumerNotFound) && !errors.Is(err, nats.ErrStreamNotFound) {
		return nil, fmt.Errorf("create pull subscription: %w", err)
	}

	if streamManager != nil {
		if streamErr := streamManager.EnsureStream(nil); streamErr != nil {
			return nil, fmt.Errorf("recover stream %s: %w", cm.streamName, streamErr)
		}
	}

	if consumerConfig == nil {
		return nil, fmt.Errorf("consumer config is required for recovery")
	}
	if err := cm.EnsureConsumer(consumerConfig); err != nil {
		return nil, fmt.Errorf("recover consumer %s: %w", cm.consumerName, err)
	}

	sub, err = cm.CreatePullSubscription()
	if err != nil {
		return nil, fmt.Errorf("create pull subscription after recovery: %w", err)
	}
	return sub, nil
}
