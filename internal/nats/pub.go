package nats

import (
	"caatsm/internal/config"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

func Publish(config *config.Config, parsedMessage interface{}) error {
	logger := watermill.NewStdLogger(false, false)
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(config.Timeouts.Server),
		nc.ReconnectWait(config.Timeouts.ReconnectWait),
	}
	jsConfig := nats.JetStreamConfig{Disabled: true}

	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         config.Nats.URL,
			NatsOptions: options,
			JetStream:   jsConfig,
		},
		logger,
	)
	if err != nil {
		panic(err)
	}

	logger.Info("NATS server connected", map[string]interface{}{"url": config.Nats.URL})
	logger.Info("Publishing message to NATS topic", map[string]interface{}{"topic": config.Publisher.Topic})

	messageText, err := json.Marshal(parsedMessage)
	if err != nil {
		logger.Error("Failed to marshal message", err, map[string]interface{}{"message": parsedMessage})
	}
	msg := message.NewMessage(watermill.NewUUID(), []byte(messageText))
	err = publisher.Publish(config.Publisher.Topic, msg)
	if err != nil {
		logger.Error("Failed to publish message to NATS topic", err, map[string]interface{}{"topic": config.Publisher.Topic})
		return err
	}
	return nil
}
