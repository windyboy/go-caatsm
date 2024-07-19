package parsers

import (
	"caatsm/internal/config"
	"regexp"
)

var bodyTypePattern = regexp.MustCompile(`^\(([A-Z]{3})(.*\n?)+\)$`)

// FindPatterns detects the message body type based on the configuration.
func FindPatterns(messageBody string, config *config.Config) *config.BodyConfig {
	if match := bodyTypePattern.FindStringSubmatch(messageBody); len(match) > 1 {
		name := match[1]
		for _, body := range config.Body {
			if body.Name == name {
				return &body
			}
		}
	}
	return nil
}

// ParseBody parses the message body and extracts the data based on the patterns defined in the configuration.
func ParseBody(messageBody string, config *config.Config) map[string]string {
	if body := FindPatterns(messageBody, config); body != nil {
		for _, pattern := range body.Patterns {
			if matches := pattern.Expression.FindStringSubmatch(messageBody); matches != nil {
				result := make(map[string]string)
				for i, name := range pattern.Expression.SubexpNames() {
					if i != 0 && name != "" {
						result[name] = matches[i]
					}
				}
				return result
			}
		}
	}
	return nil
}
