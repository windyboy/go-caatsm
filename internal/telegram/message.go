package telegram

import (
	"caatsm/internal/telegram/aviation"
	"context"
	"crypto/rand"
	"fmt"
	"time"
)

// Protocol identifies the message standard of a parsed telegram.
type Protocol string

const (
	ProtocolUnknown Protocol = "unknown"
	ProtocolAFTN    Protocol = "aftn"
	ProtocolWeather Protocol = "weather"
)

// MessageStatus represents the parsing and validation state of a telegram,
// decoupled from downstream persistence or transport failures.
type MessageStatus string

const (
	StatusUnknown     MessageStatus = "unknown"
	StatusParsed      MessageStatus = "parsed"
	StatusHeaderError MessageStatus = "header_error"
	StatusBodyError   MessageStatus = "body_error"
	StatusAFTNError   MessageStatus = "aftn_error"
)

var (
	// ErrHeaderParse indicates an invalid aviation telegram header section.
	ErrHeaderParse = aviation.ErrHeaderParse
	// ErrBodyParse indicates a failure parsing the telegram body.
	ErrBodyParse = aviation.ErrBodyParse
)

// Message is the core parsed telegram envelope shared across the pipeline.
type Message struct {
	UUID               string        `json:"uuid"`
	MessageID          string        `json:"messageId"`
	DateTime           string        `json:"dateTime"`
	PriorityIndicator  string        `json:"priorityIndicator"`
	PrimaryAddress     string        `json:"primaryAddress"`
	SecondaryAddresses string        `json:"secondaryAddresses,omitempty"`
	Originator         string        `json:"originator,omitempty"`
	OriginatorDateTime string        `json:"originatorDateTime,omitempty"`
	Protocol           Protocol      `json:"protocol,omitempty"`
	Category           string        `json:"category,omitempty"`
	Body               string        `json:"body"`
	Content            string        `json:"content,omitempty"`
	BodyData           any           `json:"bodyData,omitempty"`
	ReceivedAt         time.Time     `json:"receivedAt"`
	ParsedAt           time.Time     `json:"parsedAt,omitempty"`
	DispatchedAt       time.Time     `json:"dispatchedAt,omitempty"`
	NeedDispatch       bool          `json:"needDispatch"`
	Parsed             bool          `json:"parsed"`
	Comments           string        `json:"comments,omitempty"`
	Status             MessageStatus `json:"status"`
	ErrorReason        string        `json:"errorReason,omitempty"`
	// PublishID carries an optional deterministic idempotency key for downstream publishing
	// (e.g., inbound transport message ID or business key) and is omitted from JSON output.
	PublishID string `json:"-"`
}

// NewMessage initializes a Message with default unparsed status.
func NewMessage() *Message {
	return &Message{
		Protocol: ProtocolUnknown,
		Parsed:   false,
		Status:   StatusUnknown,
	}
}

// NewUUID generates an RFC 4122 version 4 UUID string using crypto/rand.
func NewUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type publishIDContextKey struct{}

// WithPublishID attaches a deterministic outbound publish deduplication key to ctx.
func WithPublishID(ctx context.Context, publishID string) context.Context {
	if publishID == "" {
		return ctx
	}
	return context.WithValue(ctx, publishIDContextKey{}, publishID)
}

// PublishIDFromContext extracts a deterministic outbound publish deduplication key from ctx.
func PublishIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(publishIDContextKey{}).(string); ok {
		return id
	}
	return ""
}
