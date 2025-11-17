package mapper

import (
	"caatsm/internal/adapter/dto"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TelegramMapper maps between ParsedTelegram and database rows
type TelegramMapper struct{}

// NewTelegramMapper creates a new telegram mapper
func NewTelegramMapper() *TelegramMapper {
	return &TelegramMapper{}
}

// ToDBRow converts a ParsedTelegram to a database row representation
func (m *TelegramMapper) ToDBRow(msg *dto.ParsedTelegram) ([]interface{}, error) {
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

	status := msg.Status
	if status == "" {
		status = dto.MessageStatusUnknown
	}

	return []interface{}{
		msgUUID,                // uuid
		msg.MessageID,          // message_id
		msg.DateTime,           // date_time
		msg.PriorityIndicator,  // priority_indicator
		msg.PrimaryAddress,     // primary_address
		secondaryAddresses,     // secondary_addresses (TEXT)
		msg.Originator,         // originator
		msg.OriginatorDateTime, // originator_date_time
		msg.Category,           // category
		msg.Content,            // content (TEXT, original message)
		bodyDataJSON,           // body_data (JSONB)
		string(status),         // status
		msg.ErrorReason,        // error_reason
		msg.ReceivedAt,         // received_at
		msg.ParsedAt,           // parsed_at
		msg.DispatchedAt,       // dispatched_at
		msg.NeedDispatch,       // need_dispatch
	}, nil
}

// FromDBRow converts a database row to a ParsedTelegram
func (m *TelegramMapper) FromDBRow(row []interface{}) (*dto.ParsedTelegram, error) {
	const expectedColumns = 17
	if len(row) < expectedColumns {
		return nil, fmt.Errorf("expected %d columns, got %d", expectedColumns, len(row))
	}

	msgUUID, ok := row[0].(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("column 0 must be uuid.UUID")
	}

	toString := func(v interface{}) string {
		if v == nil {
			return ""
		}
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprint(v)
	}

	parseTime := func(v interface{}) time.Time {
		if v == nil {
			return time.Time{}
		}
		if t, ok := v.(time.Time); ok {
			return t
		}
		return time.Time{}
	}

	toBool := func(v interface{}) bool {
		switch val := v.(type) {
		case bool:
			return val
		case *bool:
			return val != nil && *val
		case int64:
			return val != 0
		default:
			return false
		}
	}

	var bodyData interface{}
	if raw := row[10]; raw != nil {
		switch val := raw.(type) {
		case []byte:
			if len(val) > 0 {
				if err := json.Unmarshal(val, &bodyData); err != nil {
					return nil, fmt.Errorf("failed to unmarshal body data: %w", err)
				}
			}
		default:
			bodyData = val
		}
	}

	status := dto.MessageStatusUnknown
	if rawStatus := toString(row[11]); rawStatus != "" {
		status = dto.MessageStatus(rawStatus)
	}

	return &dto.ParsedTelegram{
		Uuid:               msgUUID.String(),
		MessageID:          toString(row[1]),
		DateTime:           toString(row[2]),
		PriorityIndicator:  toString(row[3]),
		PrimaryAddress:     toString(row[4]),
		SecondaryAddresses: toString(row[5]),
		Originator:         toString(row[6]),
		OriginatorDateTime: toString(row[7]),
		Category:           toString(row[8]),
		Content:            toString(row[9]),
		BodyData:           bodyData,
		Status:             status,
		ErrorReason:        toString(row[12]),
		ReceivedAt:         parseTime(row[13]),
		ParsedAt:           parseTime(row[14]),
		DispatchedAt:       parseTime(row[15]),
		NeedDispatch:       toBool(row[16]),
	}, nil
}
