package nats

import (
	"context"
	"fmt"
	"time"

	"caatsm/internal/config"
	"caatsm/internal/iface"
	applog "caatsm/internal/infra/log"
	"caatsm/internal/observability/metrics"

	"github.com/nats-io/nats.go"
)

type Consumer struct {
	js      nats.JetStreamContext
	config  *config.Config
	sub     *nats.Subscription
	handler iface.MessageHandler
}

func NewConsumer(js nats.JetStreamContext, cfg *config.Config, handler iface.MessageHandler) (*Consumer, error) {
	return &Consumer{
		js:      js,
		config:  cfg,
		handler: handler,
	}, nil
}

// Subscribe subscribes to a JetStream subject and processes messages
func (c *Consumer) Subscribe(ctx context.Context) error {
	log := applog.Sugared()

	subject := c.config.Subscription.Topic
	queueGroup := c.config.Subscription.QueueGroup

	// Create consumer configuration
	consumerConfig := &nats.ConsumerConfig{
		Durable:       queueGroup,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       c.config.Timeouts.AckWait,
		MaxDeliver:    5, // Max retry attempts
		FilterSubject: subject,
	}

	// Create or get consumer
	consumerInfo, err := c.js.AddConsumer(c.config.Nats.JetStream.StreamName, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	// Subscribe to messages
	sub, err := c.js.PullSubscribe(subject, queueGroup, nats.Bind(c.config.Nats.JetStream.StreamName, consumerInfo.Name))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	c.sub = sub
	log.Infof("Subscribed to JetStream subject: %s (queue: %s)", subject, queueGroup)

	// Process messages
	go c.processMessages(ctx)

	return nil
}

func (c *Consumer) processMessages(ctx context.Context) {
	log := applog.Sugared()
	batchSize := 10
	subject := c.config.Subscription.Topic

	for {
		select {
		case <-ctx.Done():
			log.Info("Context cancelled, stopping message processing")
			return
		default:
			batchStart := time.Now()
			msgs, err := c.sub.Fetch(batchSize, nats.MaxWait(5*time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				log.Errorf("Failed to fetch messages: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			metrics.WorkerBatchDuration.Observe(time.Since(batchStart).Seconds())

			for _, msg := range msgs {
				metrics.NATSMessagesConsumed.WithLabelValues(subject).Inc()
				if err := c.handleMessage(ctx, msg); err != nil {
					log.Errorf("Failed to handle message: %v", err)
					msg.Nak()
					metrics.MessageRetriesTotal.Inc()
				} else {
					msg.Ack()
				}
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg *nats.Msg) error {
	messageID := msg.Header.Get("Nats-Msg-Id")
	if messageID == "" {
		messageID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	return c.handler.HandleMessage(ctx, msg.Data, messageID)
}

// Close closes the consumer subscription
func (c *Consumer) Close() error {
	if c.sub != nil {
		return c.sub.Unsubscribe()
	}
	return nil
}
