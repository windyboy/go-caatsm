package aviation

import "regexp"

// BodyConfig represents the configuration for parsing message bodies.
type BodyConfig struct {
	Patterns []PatternConfig
}

// PatternConfig represents the configuration for a specific pattern.
type PatternConfig struct {
	Pattern    string
	Comments   string
	Expression *regexp.Regexp
}

var (
	bodyPatterns = map[string]BodyConfig{}
)

func init() {
	// Initialize body patterns.
	bodyPatterns = map[string]BodyConfig{
		"ARR": {
			Patterns: []PatternConfig{
				{
					Pattern:    ArrPatternString,
					Comments:   "Pattern for ARR message",
					Expression: ArrPatternExpression,
				},
			},
		},
		"DEP": {
			Patterns: []PatternConfig{
				{
					Pattern:    DepPatternString,
					Comments:   "Pattern for DEP message",
					Expression: DepPatternExpression,
				},
			},
		},
		"FPL": {
			Patterns: []PatternConfig{
				{
					Pattern:    FplPatternString,
					Comments:   "Pattern for FPL message",
					Expression: FplPatternExpression,
				},
			},
		},
		"CNL": {
			Patterns: []PatternConfig{
				{
					Pattern:    CnlPatternString,
					Comments:   "Pattern for CNL message",
					Expression: CnlPatternExpression,
				},
			},
		},
		"DLA": {
			Patterns: []PatternConfig{
				{
					Pattern:    DlaPatternString,
					Comments:   "Pattern for DLA message",
					Expression: DlaPatternExpression,
				},
			},
		},
	}
}

// FindPatterns finds the matching body configuration based on the message body.
func FindPatterns(messageBody string) *BodyConfig {
	if match := BodyTypePattern.FindStringSubmatch(messageBody); len(match) > 1 {
		name := match[1]
		if bodyConfig, found := bodyPatterns[name]; found {
			return &bodyConfig
		}
	}
	return nil
}

// ParseBody parses the message body and returns the extracted values.
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
