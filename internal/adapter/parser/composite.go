package parser

import (
	"caatsm/internal/adapter/dto"
	"caatsm/internal/domain/weather"
	"caatsm/internal/port"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CompositeParser combines multiple parsers (weather and aviation)
type CompositeParser struct {
	aviationParser Parser
	weatherParser  port.WeatherParser
}

// NewCompositeParser creates a new composite parser
func NewCompositeParser(aviation Parser, weather port.WeatherParser) *CompositeParser {
	return &CompositeParser{
		aviationParser: aviation,
		weatherParser:  weather,
	}
}

// Parse attempts to parse using multiple parsers
func (p *CompositeParser) Parse(rawText string) (*dto.ParsedTelegram, error) {
	// 1. Try weather parser first
	if p.weatherParser != nil && p.weatherParser.CanParse(rawText) {
		wMsg, err := p.weatherParser.Parse(rawText)
		if err == nil {
			return p.weatherToTelegram(wMsg, rawText), nil
		}
		// If parsing fails, continue to aviation parser as fallback
	}

	// 2. Try aviation parser (existing logic)
	return p.aviationParser.Parse(rawText)
}

// weatherToTelegram converts a WeatherMessage to ParsedTelegram
func (p *CompositeParser) weatherToTelegram(wMsg weather.WeatherMessage, raw string) *dto.ParsedTelegram {
	parsed := dto.NewParsedTelegram()
	parsed.Content = raw
	parsed.Body = raw
	parsed.Category = string(wMsg.Type())
	parsed.BodyData = wMsg
	parsed.Parsed = true
	parsed.Status = dto.MessageStatusParsed
	parsed.Uuid = uuid.New().String()
	parsed.ReceivedAt = time.Now()
	parsed.ParsedAt = time.Now()

	// Extract basic information from weather message
	switch msg := wMsg.(type) {
	case *weather.Metar:
		issueTime := msg.IssueTime()
		parsed.MessageID = fmt.Sprintf("%s-%s", msg.Station(), issueTime.Format("20060102150405"))
		parsed.DateTime = issueTime.Format("060102150405")
	case *weather.Taf:
		issueTime := msg.IssueTime()
		parsed.MessageID = fmt.Sprintf("%s-%s", msg.Station(), issueTime.Format("20060102150405"))
		parsed.DateTime = issueTime.Format("060102150405")
	}

	return parsed
}

