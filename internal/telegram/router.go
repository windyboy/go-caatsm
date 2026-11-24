package telegram

import (
	"caatsm/internal/telegram/aviation"
	"caatsm/internal/telegram/weather"
	"fmt"
	"time"
)

// Router routes raw telegram payloads to the appropriate protocol parser
// (weather reports vs ICAO AFTN aviation telegrams).
type Router struct {
	weather *weather.Parser
}

// NewRouter creates a composite telegram Router.
func NewRouter() *Router {
	return &Router{
		weather: weather.NewParser(),
	}
}

// Parse classifies and parses rawText into a Message.
// Always returns a non-nil *Message even on error so failed payloads can be audited.
func (r *Router) Parse(rawText string) (*Message, error) {
	if r.weather != nil && r.weather.CanParse(rawText) {
		wMsg, err := r.weather.Parse(rawText)
		if err != nil {
			now := time.Now()
			category, _ := weather.Classify(rawText)
			msg := NewMessage()
			msg.UUID = NewUUID()
			msg.Protocol = ProtocolWeather
			msg.Category = category
			msg.Content = rawText
			msg.Body = rawText
			msg.Status = StatusBodyError
			msg.ErrorReason = err.Error()
			msg.Comments = err.Error()
			msg.ReceivedAt = now
			msg.ParsedAt = now
			return msg, fmt.Errorf("%w: %w", ErrBodyParse, err)
		}
		return weatherToMessage(wMsg, rawText), nil
	}

	res, err := aviation.Parse(rawText)
	return aviationResultToMessage(res, rawText), err
}

func weatherToMessage(wMsg weather.Message, raw string) *Message {
	now := time.Now()
	issueTime := wMsg.IssueTime()

	msg := NewMessage()
	msg.UUID = NewUUID()
	msg.Protocol = ProtocolWeather
	msg.Category = string(wMsg.Type())
	msg.MessageID = fmt.Sprintf("%s-%s", wMsg.Station(), issueTime.Format("20060102150405"))
	msg.DateTime = issueTime.Format("060102150405")
	msg.Content = raw
	msg.Body = raw
	msg.BodyData = wMsg
	msg.Parsed = true
	msg.Status = StatusParsed
	msg.ReceivedAt = now
	msg.ParsedAt = now
	return msg
}

func aviationResultToMessage(res *aviation.Result, raw string) *Message {
	if res == nil {
		msg := NewMessage()
		msg.UUID = NewUUID()
		msg.Protocol = ProtocolAFTN
		msg.Content = raw
		msg.Status = StatusBodyError
		msg.ErrorReason = "aviation parser returned nil"
		return msg
	}

	msg := &Message{
		UUID:               NewUUID(),
		MessageID:          res.Header.MessageID,
		DateTime:           res.Header.DateTime,
		PriorityIndicator:  res.Header.PriorityIndicator,
		PrimaryAddress:     res.Header.PrimaryAddress,
		SecondaryAddresses: res.Header.SecondaryAddresses,
		Originator:         res.Header.Originator,
		OriginatorDateTime: res.Header.OriginatorDateTime,
		Protocol:           ProtocolAFTN,
		Category:           res.Header.Category,
		Body:               res.Header.Body,
		Content:            res.Header.Content,
		BodyData:           res.BodyData,
		ReceivedAt:         res.Header.ReceivedAt,
		ParsedAt:           res.Header.ParsedAt,
		Parsed:             res.Parsed,
		Comments:           res.Comments,
		ErrorReason:        res.ErrorReason,
	}

	switch res.Status {
	case aviation.StatusParsed:
		msg.Status = StatusParsed
	case aviation.StatusHeaderError:
		msg.Status = StatusHeaderError
	case aviation.StatusBodyError:
		msg.Status = StatusBodyError
	default:
		msg.Status = StatusUnknown
	}
	return msg
}
