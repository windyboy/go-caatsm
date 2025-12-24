package port

import "caatsm/internal/domain/weather"

// WeatherParser defines the interface for parsing weather reports
type WeatherParser interface {
	// CanParse determines if the raw string can be parsed as a weather report
	CanParse(raw string) bool

	// Parse parses a raw weather report string and returns a WeatherMessage
	Parse(raw string) (weather.WeatherMessage, error)
}

