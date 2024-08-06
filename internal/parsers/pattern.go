package parsers

import (
	"regexp"
)

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

// LineParser represents a line parser configuration.
type LineParser struct {
	Airlines      []string
	MinLen        int
	WaypointStart int
	Fields        map[int]string
}

var (
	bodyPatterns = map[string]BodyConfig{}
	parserMap    = map[string]*regexp.Regexp{}
	parserDef    = &[]LineParser{}
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

	// Initialize parser map.
	parserMap = map[string]*regexp.Regexp{
		Index:        IndexExpression,
		Task:         TaskExpression,
		Date:         DateExpression,
		FlightNumber: FlightNumberExpression,
		Register:     RegisterExpression,
	}

	// Initialize parser definitions.
	parserDef = &[]LineParser{
		{
			Airlines:      []string{"FM"},
			MinLen:        6,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Task,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"MF"},
			MinLen:        5,
			WaypointStart: 4,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"8X"},
			MinLen:        9,
			WaypointStart: 7,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"HU"},
			MinLen:        6,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"JD"},
			MinLen:        7,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"GS"},
			MinLen:        4,
			WaypointStart: 3,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"Y8"},
			MinLen:        6,
			WaypointStart: 3,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"3U"},
			MinLen:        8,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"CK"},
			MinLen:        4,
			WaypointStart: 3,
			Fields: map[int]string{
				0: Task,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"G5"},
			MinLen:        8,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"9C"},
			MinLen:        8,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Date,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"ZH"},
			MinLen:        9,
			WaypointStart: 7,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: Date,
				3: FlightNumber,
				4: Register,
			},
		},
		{
			Airlines:      []string{"8L"},
			MinLen:        6,
			WaypointStart: 4,
			Fields: map[int]string{
				0: Index,
				1: Task,
				2: FlightNumber,
				3: Register,
			},
		},
		{
			Airlines:      []string{"SC"},
			MinLen:        9,
			WaypointStart: 7,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"PN"},
			MinLen:        7,
			WaypointStart: 5,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"CZ"},
			MinLen:        6,
			WaypointStart: 4,
			Fields: map[int]string{
				0: Index,
				1: FlightNumber,
				2: Register,
			},
		},
		{
			Airlines:      []string{"HO"},
			MinLen:        7,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: Task,
				3: FlightNumber,
				4: Register,
			},
		},
		{
			Airlines:      []string{"NS"},
			MinLen:        7,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Index,
				1: Date,
				2: Task,
				3: FlightNumber,
				4: Register,
			},
		},
		{
			Airlines:      []string{"EU"},
			MinLen:        7,
			WaypointStart: 6,
			Fields: map[int]string{
				0: Task,
				1: Date,
				2: FlightNumber,
				3: Register,
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
