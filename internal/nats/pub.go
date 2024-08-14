package nats

import (
	"caatsm/internal/config"
	"caatsm/pkg/utils"
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

type NatsPublisher struct {
	config    *config.Config
	publisher *nats.Publisher
}

func NewPub(config *config.Config) *NatsPublisher {
	logger := watermill.NewStdLogger(false, false)

	jsConfig := nats.JetStreamConfig{Disabled: true}
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(config.Timeouts.Server),
		nc.ReconnectWait(config.Timeouts.ReconnectWait),
	}
	publisher, _ := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         config.Nats.URL,
			NatsOptions: options,
			JetStream:   jsConfig,
		}, logger)
	return &NatsPublisher{
		config:    config,
		publisher: publisher,
	}
}

func (n *NatsPublisher) Publish(parsedMessage interface{}) error {
	logger := utils.GetSugaredLogger()

	messageText, err := json.Marshal(parsedMessage)
	if err != nil {
		logger.Errorf("Failed to marshal message: %v", err)
	}
	msg := message.NewMessage(watermill.NewUUID(), []byte(messageText))
	err = n.publisher.Publish(n.config.Publisher.Topic, msg)
	if err != nil {
		logger.Errorf("Failed to publish message: %v", err)
		return err
	}
	return nil
}
