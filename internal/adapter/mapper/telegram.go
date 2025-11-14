package mapper

import (
	"caatsm/internal/domain"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

// TelegramMapper maps between domain.ParsedMessage and database rows
type TelegramMapper struct{}

// NewTelegramMapper creates a new telegram mapper
func NewTelegramMapper() *TelegramMapper {
	return &TelegramMapper{}
}

// ToDBRow converts a domain.ParsedMessage to a database row representation
func (m *TelegramMapper) ToDBRow(msg *domain.ParsedMessage) ([]interface{}, error) {
	// Parse UUID
	var msgUUID uuid.UUID
	var err error
	if msg.Uuid != "" {
		msgUUID, err = uuid.Parse(msg.Uuid)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID: %w", err)
		}
	} else {
		msgUUID = uuid.New()
	}

	// Marshal BodyData to JSONB
	var bodyDataJSON []byte
	if msg.BodyData != nil {
		bodyDataJSON, err = json.Marshal(msg.BodyData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body data: %w", err)
		}
	}

	// SecondaryAddresses is already a string, so we can use it directly
	secondaryAddresses := msg.SecondaryAddresses

	return []interface{}{
		msgUUID,                    // uuid
		msg.MessageID,              // message_id
		msg.DateTime,               // date_time
		msg.PriorityIndicator,      // priority_indicator
		msg.PrimaryAddress,         // primary_address
		secondaryAddresses,         // secondary_addresses (TEXT)
		msg.Originator,             // originator
		msg.OriginatorDateTime,     // originator_date_time
		msg.Category,               // category
		msg.Content,                // content (TEXT, original message)
		bodyDataJSON,               // body_data (JSONB)
		msg.ReceivedAt,             // received_at
		msg.ParsedAt,               // parsed_at
		msg.DispatchedAt,           // dispatched_at
		msg.NeedDispatch,           // need_dispatch
	}, nil
}

// FromDBRow converts a database row to a domain.ParsedMessage
func (m *TelegramMapper) FromDBRow(row []interface{}) (*domain.ParsedMessage, error) {
	// This is a placeholder - will be implemented if needed for queries
	// For now, we only need ToDBRow for inserts
	return nil, fmt.Errorf("FromDBRow not implemented")
}

