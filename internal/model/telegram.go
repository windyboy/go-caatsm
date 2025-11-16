package model

import (
	"time"
)

// MessageStatus represents the parsing status of a telegram.
// It is intentionally decoupled from infrastructure concerns (e.g. DB or publish failures)
// so that domain parsing state can be reasoned about independently.
type MessageStatus string

const (
	MessageStatusUnknown     MessageStatus = "unknown"
	MessageStatusParsed      MessageStatus = "parsed"
	MessageStatusHeaderError MessageStatus = "header_error"
	MessageStatusBodyError   MessageStatus = "body_error"
)

// ParsedTelegram holds the parsed data from an aviation message.
// It is a transport-oriented model used by the application pipeline (parser,
// persistence, publishing), and may embed domain-specific body structures
// (e.g. *domain.FPL, *domain.DEP) in BodyData.
type ParsedTelegram struct {
	Uuid               string      `json:"uuid"`
	MessageID          string      `json:"messageId"`
	DateTime           string      `json:"dateTime"`
	PriorityIndicator  string      `json:"priorityIndicator"`
	PrimaryAddress     string      `json:"primaryAddress"`
	SecondaryAddresses string      `json:"secondaryAddresses,omitempty"`
	Originator         string      `json:"originator,omitempty"`
	OriginatorDateTime string      `json:"originatorDateTime,omitempty"`
	Category           string      `json:"category,omitempty"`
	Body               string      `json:"body"`
	Content            string      `json:"content,omitempty"`
	BodyData           interface{} `json:"bodyData,omitempty"`
	ReceivedAt         time.Time   `json:"receivedAt"`
	ParsedAt           time.Time   `json:"parsedAt,omitempty"`
	DispatchedAt       time.Time   `json:"dispatchedAt,omitempty"`
	NeedDispatch       bool        `json:"needDispatch"`
	Parsed             bool        `json:"parsed"`
	Comments           string      `json:"comments,omitempty"`
	Status             MessageStatus
	ErrorReason        string `json:"errorReason,omitempty"`
}

// NewParsedTelegram initializes a ParsedTelegram with default values.
func NewParsedTelegram() *ParsedTelegram {
	return &ParsedTelegram{
		Parsed: false,
		Status: MessageStatusUnknown,
	}
}
