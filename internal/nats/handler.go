package nats

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/internal/iface"
	"caatsm/internal/parsers"
	"caatsm/pkg/utils"
	"fmt"
	"sync"
)

type MessageHandler struct {
	mu         sync.Mutex
	config     *config.Config
	repository iface.MessageRepository
	publisher  iface.MessagePublisher
}

func NewHandler(config *config.Config, publisher iface.MessagePublisher, repository iface.MessageRepository) *MessageHandler {
	return &MessageHandler{
		config:     config,
		repository: repository,
		publisher:  publisher,
	}
}

func (handler *MessageHandler) HandleMessage(msg []byte, id string) error {
	handler.mu.Lock()
	defer handler.mu.Unlock()
	log := utils.GetSugaredLogger()
	if msg == nil {
		log.Error("empty message")
		return fmt.Errorf("empty message")
	}
	payload := string(msg)
	var parsed *domain.ParsedMessage
	if parsed = parsers.Parse(payload); !parsed.Parsed {
		log.Infof("not parsed: [%s] : {%s} \n", id, payload)
	} else {
		parsed.Uuid = id
		log.Infof("parsed [%s]: %v\n", id, parsed.ToString())
	}
	handler.repository.CreateNew(parsed)

	handler.publisher.Publish(parsed)

	return nil
}
