package parser

import (
	"caatsm/internal/adapter/parser/aviation"
	"caatsm/internal/port"
)

// ProvideParser creates a composite parser instance that combines weather and aviation parsers.
func ProvideParser(weatherParser port.WeatherParser) Parser {
	aviationParser := aviation.NewParser()
	return NewCompositeParser(aviationParser, weatherParser)
}
