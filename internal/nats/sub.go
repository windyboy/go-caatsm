package nats // Already has package comment

import (
	"caatsm/internal/config"
	"caatsm/internal/iface"
	"caatsm/pkg/utils"
	"context"
	"errors"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NatsSubscriber handles subscribing to NATS topics and processing messages using Watermill.
type NatsSubscriber struct {
	config     *config.Config    // config holds application configuration, particularly NATS and subscription settings.
	subscriber *nats.Subscriber // subscriber is the Watermill NATS subscriber instance.
}

// NewSub creates and initializes a new NatsSubscriber.
// It configures a Watermill NATS subscriber based on the provided application configuration.
// config: The application configuration.
// Returns a pointer to the initialized NatsSubscriber.
// Note: The current implementation uses `subscriber, _ := nats.NewSubscriber(...)` which discards
// the error from `nats.NewSubscriber`. This should be handled in a production environment.
func NewSub(config *config.Config) *NatsSubscriber {
	logger := watermill.NewStdLogger(false, false) // Watermill logger; can be configured (e.g., to use utils.Logger).
	marshaler := &PlainTextMarshaler{}             // Custom marshaler to handle raw byte payloads.

	// NATS connection options from the application configuration.
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(config.Timeouts.Server),
		nc.ReconnectWait(config.Timeouts.ReconnectWait),
	}
	
	// JetStream is disabled by default in this configuration.
	jsConfig := nats.JetStreamConfig{Disabled: true}

	// Create a new Watermill NATS subscriber.
	// Error from nats.NewSubscriber is currently ignored with `_`.
	// In a production system, this error should be checked and handled.
	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:            config.Nats.URL,
			CloseTimeout:   config.Timeouts.Close,
			AckWaitTimeout: config.Timeouts.AckWait, // Used for NATS JetStream acknowledgment.
			NatsOptions:    options,
			Unmarshaler:    marshaler, // Custom unmarshaler for message.Message payload.
			JetStream:      jsConfig,
		},
		logger,
	)
	if err != nil {
		utils.GetSugaredLogger().Fatalf("Failed to create NATS subscriber: %v", err) // Added Fatalf
	}

	return &NatsSubscriber{
		config:     config,
		subscriber: subscriber,
	}
}

// Subscribe connects to the configured NATS topic and starts processing messages.
// It uses the provided iface.MessageHandler to handle each received message.
// Messages are acknowledged upon successful handling by the handler; otherwise, they are Nacked.
// This method will block as it ranges over the message channel.
// It ensures the subscriber is closed when the message channel closes or an error occurs.
// handlers: An implementation of iface.MessageHandler to process messages.
func (n *NatsSubscriber) Subscribe(handlers iface.MessageHandler) {
	logger := utils.GetSugaredLogger()

	defer func() {
		if err := n.subscriber.Close(); err != nil {
			logger.Errorf("NatsSubscriber: Error closing subscriber: %v", err)
		} else {
			logger.Info("NatsSubscriber: Subscriber closed successfully.")
		}
	}()
	
	// Subscribe to the topic.
	topic := n.config.Subscription.Topic
	if topic == "" {
		logger.Fatalf("NatsSubscriber: Subscription topic is not configured.") // Use Fatalf as this is a critical config error
	}

	messages, err := n.subscriber.Subscribe(context.Background(), topic)
	if err != nil {
		logger.Errorf("NatsSubscriber: Failed to subscribe to topic '%s': %v", topic, err)
		return // Exit if subscription fails
	}
	logger.Infof("NatsSubscriber: Successfully subscribed to topic '%s'. Waiting for messages...", topic)

	// Process incoming messages.
	for msg := range messages {
		logger.Debugf("NatsSubscriber: Received message ID: %s on topic '%s'", msg.UUID, topic)
		if err := handlers.HandleMessage(msg.Payload, msg.UUID); err == nil {
			logger.Infof("NatsSubscriber: Message handled successfully, ID: %s. Sending ACK.", msg.UUID)
			msg.Ack() // Acknowledge the message after successful processing.
		} else {
			// If HandleMessage returns an error, it means publishing failed.
			// The message was likely still parsed and stored by the handler.
			logger.Errorf("NatsSubscriber: Handler failed to process message ID: %s (error: %v). Sending NACK.", msg.UUID, err)
			msg.Nack() // Negative acknowledgment; message might be redelivered depending on NATS server config.
		}
	}
	// This point is reached if the messages channel is closed, e.g., by subscriber.Close().
	logger.Info("NatsSubscriber: Message channel closed, subscription ending.")
}

// PlainTextMarshaler is a custom Watermill Unmarshaler.
// It assumes the NATS message data is the raw payload to be used directly
// in a Watermill message, without any specific marshalling format like JSON envelope.
type PlainTextMarshaler struct{}

// Marshal is part of the nats.Unmarshaler interface.
// For this subscriber, it's not used as we are unmarshalling from NATS, not marshalling to NATS via this path.
// However, a conforming implementation is required. It simply returns the data as is.
func (m *PlainTextMarshaler) Marshal(topic string, msg nc.Msg) ([]byte, error) {
	return msg.Data, nil
}

// Unmarshal converts a NATS message (`nc.Msg`) into a Watermill `message.Message`.
// It assigns a new UUID to the Watermill message and uses the NATS message's data as payload.
func (m *PlainTextMarshaler) Unmarshal(natsMsg *nc.Msg) (*message.Message, error) {
	if natsMsg == nil {
		return nil, errors.New("cannot unmarshal nil NATS message")
	}
	// Create a new Watermill message. The UUID is generated by Watermill.
	// The payload is the raw data from the NATS message.
	watermillMsg := message.NewMessage(watermill.NewUUID(), natsMsg.Data)
	// Metadata could be copied if needed:
	// e.g., watermillMsg.Metadata.Set("subject", natsMsg.Subject)
	return watermillMsg, nil
}
