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
