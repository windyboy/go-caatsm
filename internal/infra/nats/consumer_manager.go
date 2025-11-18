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

// RecoverResources attempts to recreate the stream and consumer in dev/test environments
func (cm *ConsumerManager) RecoverResources(streamManager *StreamManager, consumerConfig *nats.ConsumerConfig) error {
	if cm.js == nil {
		return fmt.Errorf("jetstream context is nil")
	}

	// Ensure stream exists (dev/test may auto-create, prod will error).
	if err := streamManager.EnsureStream(); err != nil {
		return fmt.Errorf("ensure stream %s: %w", cm.streamName, err)
	}

	// Ensure durable consumer exists and is properly bound.
	if err := cm.EnsureConsumer(consumerConfig); err != nil {
		return fmt.Errorf("ensure consumer %s: %w", cm.consumerName, err)
	}

	return nil
}

// CreatePullSubscription creates a pull subscription with recovery logic
func (cm *ConsumerManager) CreatePullSubscription() (*nats.Subscription, error) {
	return cm.js.PullSubscribe(cm.subject, cm.consumerName, nats.Bind(cm.streamName, cm.consumerName))
}

// CreatePullSubscriptionWithRecovery creates a pull subscription and attempts recovery if needed
func (cm *ConsumerManager) CreatePullSubscriptionWithRecovery(streamManager *StreamManager, consumerConfig *nats.ConsumerConfig) (*nats.Subscription, error) {
	sub, err := cm.CreatePullSubscription()
	if err == nil {
		return sub, nil
	}

	if isJetStreamResourceNotFound(err) && isDevLikeEnv() && shouldBootstrapStream() {
		cm.logger.Warn("PullSubscribe failed due to missing JetStream resources; attempting to recreate",
			zap.Error(err),
			zap.String("stream", cm.streamName),
			zap.String("consumer", cm.consumerName),
		)
		if recErr := cm.RecoverResources(streamManager, consumerConfig); recErr != nil {
			return nil, fmt.Errorf("failed to recover JetStream resources: %w", recErr)
		}
		// Retry subscription after successful recovery.
		sub, err = cm.CreatePullSubscription()
		if err != nil {
			return nil, fmt.Errorf("failed to create pull subscription after recovery: %w", err)
		}
		return sub, nil
	}

	return nil, fmt.Errorf("failed to create pull subscription: %w", err)
}
