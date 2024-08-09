package handlers

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/internal/parsers"
	"caatsm/internal/repository"
	"caatsm/pkg/utils"
	"errors"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

type MessageHandler struct {
	mu         sync.Mutex
	config     *config.Config
	hasuraRepo *repository.HasuraRepository
}

func New(config *config.Config) *MessageHandler {
	return &MessageHandler{
		config:     config,
		hasuraRepo: repository.New(config.Hasura.Endpoint, config.Hasura.Secret),
	}
}

func (handler *MessageHandler) HandleMessage(msg *message.Message) error {
	handler.mu.Lock()
	defer handler.mu.Unlock()
	log := utils.GetSugaredLogger()
	if msg.Payload == nil {
		log.Error("empty message")
		return fmt.Errorf("empty message")
	}
	payload := string(msg.Payload)
	var parsed *domain.ParsedMessage
	if parsed = parsers.Parse(payload); !parsed.Parsed {
		log.Infof("not parsed: [%s] : {%s} \n", msg.UUID, msg.Payload)
	} else {
		log.Infof("parsed [%s]: %v\n", msg.UUID, parsed)
	}
	handler.SaveMessage(parsed, msg.UUID)
	return nil
}

func (n *MessageHandler) SaveMessage(parsed *domain.ParsedMessage, uuid string) {
	logger := watermill.NewStdLogger(false, false)
	if parsed != nil {
		parsed.Uuid = uuid
		if err := n.hasuraRepo.CreateNew(parsed); err != nil {
			logger.Error("error inserting message", err, map[string]interface{}{"message": parsed})
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
