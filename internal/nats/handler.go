// Package nats provides functionality for interacting with NATS messaging system,
// including publishing messages, subscribing to topics, and handling received messages.
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

// MessageHandler processes incoming messages received from NATS subscriptions.
// It uses a repository to store message data and a publisher to potentially forward or reply to messages.
type MessageHandler struct {
	mu         sync.Mutex                // mu is a mutex to ensure thread-safe access to handler state, if any, and to serialize message processing.
	config     *config.Config            // config holds application configuration, possibly used for message handling logic.
	repository iface.MessageRepository // repository provides an interface for data persistence operations.
	publisher  iface.MessagePublisher  // publisher provides an interface for publishing messages.
}

// NewHandler creates and returns a new MessageHandler.
// config: Application configuration.
// publisher: An instance of MessagePublisher for publishing messages.
// repository: An instance of MessageRepository for data persistence.
func NewHandler(config *config.Config, publisher iface.MessagePublisher, repository iface.MessageRepository) *MessageHandler {
	return &MessageHandler{
		config:     config,
		repository: repository,
		publisher:  publisher,
	}
}

// HandleMessage processes a single message received from NATS.
// It parses the message payload, stores it using the repository, and then publishes it.
// msg: The raw byte payload of the message.
// id: A unique identifier for the message (e.g., NATS message ID or a UUID).
// Returns an error if publishing the message fails, or if the input message is nil.
// Parsing or repository errors are logged but not returned as errors from this method,
// allowing the message to still be published if possible.
func (handler *MessageHandler) HandleMessage(msg []byte, id string) error {
	// Lock the handler to ensure that message processing is serialized for this handler instance.
	// This prevents potential race conditions if the handler's internal state were to be modified
	// concurrently or if the underlying repository/publisher are not goroutine-safe.
	// Note: This serializes all message handling; if performance is critical and I/O operations
	// (repository, publisher) are slow, this lock could become a bottleneck.
	handler.mu.Lock()
	defer handler.mu.Unlock()

	log := utils.GetSugaredLogger().With("message_id", id) // Add message_id to all log entries from this point

	log.Info("Starting message handling")

	if msg == nil {
		log.Error("Received an empty message payload")
		return fmt.Errorf("empty message with id %s", id)
	}

	payload := string(msg)
	log.Debugf("Raw message payload: %s", payload)

	// Attempt to parse the raw payload.
	// parsers.Parse is expected to always return a *domain.ParsedMessage.
	// If parsing fails, parsed.Parsed will be false, and parsed.Comments may contain error details.
	parsedMessage := parsers.Parse(payload)
	parsedMessage.Uuid = id // Assign the NATS message ID as our internal UUID

	if !parsedMessage.Parsed {
		// Log the original payload if parsing failed, along with any comments from the parser.
		log.Warnf("Message parsing failed. Parser comments: '%s'. Original payload: %s", parsedMessage.Comments, payload)
	} else {
		log.Infof("Message parsed successfully: %s", parsedMessage.ToString())
	}

	// Store the parsing result (successful or not) in the repository.
	// TODO: The CreateNew method should ideally return an error. Assuming it might for now.
	log.Debug("Attempting to save message to repository")
	if err := handler.repository.CreateNew(parsedMessage); err != nil {
		log.Errorf("Failed to create new message in repository: %v. Parsed data: %s", err, parsedMessage.ToString())
		// Depending on requirements, we might want to return err here and not publish.
		// For now, proceeding to publish as per original logic.
	} else {
		log.Debug("Message saved to repository successfully")
	}

	// Publish the ParsedMessage to a NATS subject or other messaging system.
	// This allows other services to react to the message, whether it was successfully parsed or not.
	// Consumers of this published message should check the `Parsed` field and `Comments`.
	log.Debug("Attempting to publish message")
	if err := handler.publisher.Publish(parsedMessage); err != nil {
		log.Errorf("Failed to publish message: %v. Parsed data: %s", err, parsedMessage.ToString())
		return fmt.Errorf("failed to publish message with id %s: %w", id, err) // Propagate publish error
	}
	log.Info("Message published successfully")

	log.Info("Message handling completed")
	return nil
}
