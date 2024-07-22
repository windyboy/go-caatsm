package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/parsers"
	"caatsm/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

type NatsHandler struct {
	config     *config.Config
	hasuraRepo *repository.HasuraRepository
}

func NewNatsHandler(config *config.Config) *NatsHandler {
	return &NatsHandler{config: config, hasuraRepo: repository.NewHasuraRepo(config.Hasura.Endpoint, config.Hasura.Secret)}
}

func (n *NatsHandler) Subscribe() {
	marshaler := &PlainTextMarshaler{}
	logger := watermill.NewStdLogger(false, false)
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(n.config.Timeouts.Server),
		nc.ReconnectWait(n.config.Timeouts.ReconnectWait),
	}
	jsConfig := nats.JetStreamConfig{Disabled: true}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:            n.config.Nats.URL,
			CloseTimeout:   n.config.Timeouts.Close,
			AckWaitTimeout: n.config.Timeouts.AckWait,
			NatsOptions:    options,
			Unmarshaler:    marshaler,
			JetStream:      jsConfig,
		},
		logger,
	)
	if err != nil {
		panic(err)
	}
	logger.Info("Subscribing to NATS topic", map[string]interface{}{"topic": n.config.Subscription.Topic})

	defer subscriber.Close()
	messages, err := subscriber.Subscribe(context.Background(), n.config.Subscription.Topic)
	if err != nil {
		logger.Error("Failed to subscribe to NATS topic", err, map[string]interface{}{"topic": n.config.Subscription.Topic})
		return
	}
	for msg := range messages {
		n.handleMessage(msg)
		msg.Ack()
	}
}
func (n *NatsHandler) handleMessage(msg *message.Message) error {
	// log := utils.Logger
	if msg.Payload == nil {
		fmt.Println("empty message")
		return fmt.Errorf("empty message")
	}
	payload := string(msg.Payload)
	if parsed, err := parsers.Parse(payload); err != nil {
		// log.Error("error parsing message", err, map[string]interface{}{"payload": payload})
		parsed.Uuid = msg.UUID
		fmt.Print("error parsing message", err)
		return err
	} else {
		// log.Info("message ", map[string]interface{}{"message": parsed})
		fmt.Printf("message [%s]: %v\n", parsed.Uuid, parsed)
		if err := n.hasuraRepo.InsertParsedMessage(parsed); err != nil {
			fmt.Print("error inserting message", err)
		}
	}
	return nil
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
