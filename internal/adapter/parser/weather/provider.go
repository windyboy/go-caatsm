package weather

import (
	"caatsm/internal/domain/weather"
	"caatsm/internal/port"
	"fmt"
)

// WeatherParserImpl implements the WeatherParser interface
type WeatherParserImpl struct{}

// NewWeatherParser creates a new weather parser instance
func NewWeatherParser() port.WeatherParser {
	return &WeatherParserImpl{}
}

// CanParse determines if the raw string can be parsed as a weather report
func (p *WeatherParserImpl) CanParse(raw string) bool {
	reportType, ok := Classify(raw)
	if !ok {
		return false
	}

	// Check for valid ending
	if !HasValidEnding(raw) {
		return false
	}

	_ = reportType // Suppress unused variable warning
	return true
}

// Parse parses a raw weather report string and returns a WeatherMessage
func (p *WeatherParserImpl) Parse(raw string) (weather.WeatherMessage, error) {
	reportType, ok := Classify(raw)
	if !ok {
		return nil, weather.ErrInvalidFormat
	}

	switch reportType {
	case "METAR", "SPECI":
		return parseMetar(raw, reportType)
	case "TAF":
		return parseTaf(raw)
	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}
}

