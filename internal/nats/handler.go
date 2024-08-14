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

func (handler *MessageHandler) HandleMessage(msg []byte, uuid []byte) error {
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
		log.Infof("not parsed: [%s] : {%s} \n", uuid, payload)
	} else {
		log.Infof("parsed [%s]: %v\n", uuid, parsed.ToString())
	}
	handler.repository.CreateNew(parsed, uuid)
	handler.publisher.Publish(parsed)
	return nil
}

// func (n *MessageHandler) SaveMessage(parsed *domain.ParsedMessage, uuid string) {
// 	logger := watermill.NewStdLogger(false, false)
// 	if parsed != nil {
// 		parsed.Uuid = uuid
// 		if err := n.hasuraRepo.CreateNew(parsed); err != nil {
// 			logger.Error("error inserting message", err, map[string]interface{}{"message": parsed})
// 		}
// 		logger.Info("message inserted", map[string]interface{}{"message": parsed.Uuid})
// 	}
// }

// func (n *MessageHandler) Publish(parsed *domain.ParsedMessage) {
// 	if err := Publish(n.config, parsed); err != nil {
// 		utils.GetSugaredLogger().Error("error publishing message", err, map[string]interface{}{"message": parsed})
// 	}
// }
