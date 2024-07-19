package parsers

import (
	"caatsm/internal/config"
	"regexp"
)

var (
	bodyTypePattern = regexp.MustCompile(`^\(([A-Z]{3})(.*\n?)+\)$`)
)

func FindPatterns(messageBody string) *config.BodyConfig {
	if match := bodyTypePattern.FindStringSubmatch(messageBody); len(match) > 1 {
		name := match[1]
		patters := config.GetBodyPatterns()
		if body, found := patters[name]; found {
			return &body
		}
	}
	return nil
}

func ParseBody(messageBody string) map[string]string {
	if body := FindPatterns(messageBody); body != nil {
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
