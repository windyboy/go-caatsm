package dto

import (
	"time"

	"caatsm/internal/domain"
)

// TelegramResponse represents a telegram message in API responses
type TelegramResponse struct {
	UUID               string                 `json:"uuid"`
	MessageID          string                 `json:"messageId"`
	DateTime           string                 `json:"dateTime"`
	PriorityIndicator  string                 `json:"priorityIndicator"`
	PrimaryAddress     string                 `json:"primaryAddress"`
	SecondaryAddresses string                 `json:"secondaryAddresses,omitempty"`
	Originator         string                 `json:"originator,omitempty"`
	OriginatorDateTime string                 `json:"originatorDateTime,omitempty"`
	Category           string                 `json:"category,omitempty"`
	Content            string                 `json:"content,omitempty"`
	BodyData           interface{}            `json:"bodyData,omitempty"`
	ReceivedAt         time.Time              `json:"receivedAt"`
	ParsedAt           *time.Time             `json:"parsedAt,omitempty"`
	DispatchedAt       *time.Time             `json:"dispatchedAt,omitempty"`
	NeedDispatch       bool                   `json:"needDispatch"`
	Parsed             bool                   `json:"parsed"`
	Comments           string                 `json:"comments,omitempty"`
}

// ToTelegramResponse converts a domain.ParsedMessage to TelegramResponse
func ToTelegramResponse(msg *domain.ParsedMessage) *TelegramResponse {
	if msg == nil {
		return nil
	}

	resp := &TelegramResponse{
		UUID:               msg.Uuid,
		MessageID:          msg.MessageID,
		DateTime:           msg.DateTime,
		PriorityIndicator:  msg.PriorityIndicator,
		PrimaryAddress:     msg.PrimaryAddress,
		SecondaryAddresses: msg.SecondaryAddresses,
		Originator:         msg.Originator,
		OriginatorDateTime: msg.OriginatorDateTime,
		Category:           msg.Category,
		Content:            msg.Content,
		BodyData:           msg.BodyData,
		ReceivedAt:         msg.ReceivedAt,
		NeedDispatch:       msg.NeedDispatch,
		Parsed:             msg.Parsed,
		Comments:           msg.Comments,
	}

	if !msg.ParsedAt.IsZero() {
		resp.ParsedAt = &msg.ParsedAt
	}
	if !msg.DispatchedAt.IsZero() {
		resp.DispatchedAt = &msg.DispatchedAt
	}

	return resp
}

// ListTelegramsRequest represents the request parameters for listing telegrams
type ListTelegramsRequest struct {
	Category          string    `query:"category"`
	MessageID         string    `query:"messageId"`
	PrimaryAddress    string    `query:"primaryAddress"`
	PriorityIndicator string    `query:"priorityIndicator"`
	StartTime         time.Time `query:"startTime"`
	EndTime           time.Time `query:"endTime"`
	Limit             int       `query:"limit"`
	Offset            int       `query:"offset"`
	OrderBy           string    `query:"orderBy"`
	OrderDirection    string    `query:"orderDirection"`
}

// ListTelegramsResponse represents the response for listing telegrams
type ListTelegramsResponse struct {
	Telegrams []*TelegramResponse `json:"telegrams"`
	Total     int64               `json:"total"`
	Limit     int                 `json:"limit"`
	Offset    int                 `json:"offset"`
}

