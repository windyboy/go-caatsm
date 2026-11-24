package postgres

import (
	"caatsm/internal/telegram"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var telegramColumns = []string{
	"uuid", "message_id", "date_time", "priority_indicator", "primary_address",
	"secondary_addresses", "originator", "originator_date_time", "category",
	"content", "body_data", "status", "error_reason",
	"received_at", "parsed_at", "dispatched_at", "need_dispatch",
}

type telegramRow struct {
	uuid               uuid.UUID
	messageID          string
	dateTime           string
	priorityIndicator  string
	primaryAddress     string
	secondaryAddresses string
	originator         string
	originatorDateTime string
	category           string
	content            string
	bodyDataJSON       []byte
	status             string
	errorReason        string
	receivedAt         time.Time
	parsedAt           time.Time
	dispatchedAt       time.Time
	needDispatch       bool
}

func (r telegramRow) values() []any {
	return []any{
		r.uuid,
		r.messageID,
		r.dateTime,
		r.priorityIndicator,
		r.primaryAddress,
		r.secondaryAddresses,
		r.originator,
		r.originatorDateTime,
		r.category,
		r.content,
		r.bodyDataJSON,
		r.status,
		r.errorReason,
		r.receivedAt,
		r.parsedAt,
		r.dispatchedAt,
		r.needDispatch,
	}
}

func toDBRow(msg *telegram.Message) ([]any, error) {
	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	var msgUUID uuid.UUID
	var err error
	if msg.UUID != "" {
		msgUUID, err = uuid.Parse(msg.UUID)
		if err != nil {
			return nil, fmt.Errorf("invalid UUID: %w", err)
		}
	} else {
		msgUUID = uuid.New()
	}

	var bodyDataJSON []byte
	if msg.BodyData != nil {
		bodyDataJSON, err = json.Marshal(msg.BodyData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body data: %w", err)
		}
	}

	status := msg.Status
	if status == "" {
		status = telegram.StatusUnknown
	}

	row := telegramRow{
		uuid:               msgUUID,
		messageID:          msg.MessageID,
		dateTime:           msg.DateTime,
		priorityIndicator:  msg.PriorityIndicator,
		primaryAddress:     msg.PrimaryAddress,
		secondaryAddresses: msg.SecondaryAddresses,
		originator:         msg.Originator,
		originatorDateTime: msg.OriginatorDateTime,
		category:           msg.Category,
		content:            msg.Content,
		bodyDataJSON:       bodyDataJSON,
		status:             string(status),
		errorReason:        msg.ErrorReason,
		receivedAt:         msg.ReceivedAt,
		parsedAt:           msg.ParsedAt,
		dispatchedAt:       msg.DispatchedAt,
		needDispatch:       msg.NeedDispatch,
	}
	return row.values(), nil
}

func fromDBRow(row []any) (*telegram.Message, error) {
	expectedColumns := len(telegramColumns)
	if len(row) < expectedColumns {
		return nil, fmt.Errorf("expected %d columns, got %d", expectedColumns, len(row))
	}

	msgUUID, ok := row[0].(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("column 0 must be uuid.UUID")
	}

	toString := func(v any) string {
		if v == nil {
			return ""
		}
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprint(v)
	}

	parseTime := func(v any) time.Time {
		if t, ok := v.(time.Time); ok {
			return t
		}
		return time.Time{}
	}

	toBool := func(v any) bool {
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

	var bodyData any
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

	status := telegram.StatusUnknown
	if rawStatus := toString(row[11]); rawStatus != "" {
		status = telegram.MessageStatus(rawStatus)
	}

	return &telegram.Message{
		UUID:               msgUUID.String(),
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
