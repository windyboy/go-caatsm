package nats

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

type NatsSubscriber struct {
	config     *config.Config
	subscriber *nats.Subscriber
}

func NewSub(config *config.Config) *NatsSubscriber {
	logger := watermill.NewStdLogger(false, false)
	marshaler := &PlainTextMarshaler{}
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(config.Timeouts.Server),
		nc.ReconnectWait(config.Timeouts.ReconnectWait),
	}
	jsConfig := nats.JetStreamConfig{Disabled: true}
	subscriber, _ := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:            config.Nats.URL,
			CloseTimeout:   config.Timeouts.Close,
			AckWaitTimeout: config.Timeouts.AckWait,
			NatsOptions:    options,
			Unmarshaler:    marshaler,
			JetStream:      jsConfig,
		},
		logger,
	)
	return &NatsSubscriber{
		config:     config,
		subscriber: subscriber,
	}
}

func (n *NatsSubscriber) Subscribe(config *config.Config, handlers iface.MessageHandler) {
	logger := utils.GetSugaredLogger()

	defer n.subscriber.Close()
	messages, err := n.subscriber.Subscribe(context.Background(), config.Subscription.Topic)
	if err != nil {
		logger.Errorf("Failed to subscribe to topic: %v", err)
		return
	}

	for msg := range messages {
		if err := handlers.HandleMessage(msg.Payload, []byte(msg.UUID)); err == nil {
			msg.Ack()
		} else {
			logger.Error("Failed to handle message", err, map[string]interface{}{"message": msg})
			msg.Nack()
		}
	}
}

type PlainTextMarshaler struct{}

func (m *PlainTextMarshaler) Marshal(topic string, msg nc.Msg) ([]byte, error) {
	return msg.Data, nil
}

func (m *PlainTextMarshaler) Unmarshal(newMsg *nc.Msg) (*message.Message, error) {
	if newMsg == nil {
		return nil, errors.New("empty message")
	}
	msg := message.NewMessage(watermill.NewUUID(), newMsg.Data)
	return msg, nil
}
