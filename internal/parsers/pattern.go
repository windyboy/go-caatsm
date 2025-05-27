package parsers // Package comment already added

import (
	"regexp"
)

// BodyConfig defines the structure for configuring message body parsing patterns for a specific message category.
// It holds a collection of individual pattern configurations.
type BodyConfig struct {
	Patterns []PatternConfig // Patterns is a slice of PatternConfig, each defining a regex for a part of the message body.
}

// PatternConfig defines a single regular expression pattern's configuration.
// It includes the pattern string, any comments describing it, and the compiled regex.
type PatternConfig struct {
	Pattern    string         // Pattern is the raw regular expression string.
	Comments   string         // Comments provide a human-readable description or context for the pattern.
	Expression *regexp.Regexp // Expression is the compiled form of the Pattern string, ready for use in matching.
}

// LineParser defines a configuration for parsing fixed-format lines, often used in schedule (PLN) messages.
// It specifies airline codes it applies to, minimum line length, starting position of waypoint data,
// and a map defining how to extract fields based on their position.
type LineParser struct {
	Airlines      []string       // Airlines is a list of airline codes (e.g., "FM", "MF") this parser configuration applies to.
	MinLen        int            // MinLen is the minimum expected length of a line for this parser to apply.
	WaypointStart int            // WaypointStart is the starting index (or field position) where waypoint data begins.
	Fields        map[int]string // Fields maps column/field indices to their corresponding data keys (e.g., FlightNumber, Register).
}

// Package-level variables holding parser configurations.
// These are initialized in the init() function.
var (
	// bodyPatterns maps message category strings (e.g., "ARR", "FPL") to their BodyConfig,
	// which contains regex patterns for parsing standard aviation message bodies.
	bodyPatterns = map[string]BodyConfig{}

	// parserMap maps string keys (like Date, FlightNumber) to compiled regular expressions
	// used for parsing specific fields, often within schedule (PLN) messages.
	parserMap = map[string]*regexp.Regexp{}

	// parserDef is a slice of LineParser configurations, primarily used for parsing
	// airline-specific flight schedule (PLN) message formats.
	parserDef = &[]LineParser{}
)

// init initializes the package-level parser configuration variables (bodyPatterns, parserMap, parserDef).
// It populates these maps and slices with predefined regular expressions and line parsing rules
// for various message types and airline formats. This pre-computation ensures that parsers
// are ready for use when the application starts.
func init() {
	// Initialize bodyPatterns for standard aviation messages (ARR, DEP, FPL, CNL, DLA).
	// Each category is mapped to a BodyConfig containing specific regex patterns.
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
	// Initialize parserDef with LineParser configurations for various airlines.
	// This defines how to parse specific, often fixed-format, lines from schedule messages (PLN).
	parserDef = &[]LineParser{
		// Example configuration for airline "FM"
		{
			Airlines:      []string{"FM"}, // Applies to FM airline codes
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

// FindPatterns attempts to find a BodyConfig that matches the given message body.
// It uses the BodyTypePattern regex (which typically looks for "(CAT..." at the start of the body)
// to identify the message category (e.g., "ARR", "FPL").
// If a category is identified and a corresponding BodyConfig exists in the bodyPatterns map,
// a pointer to that BodyConfig is returned.
// messageBody: The raw string of the message body.
// Returns a pointer to the matching BodyConfig if found, otherwise nil.
func FindPatterns(messageBody string) *BodyConfig {
	// BodyTypePattern is a precompiled regex, e.g., `^\(([A-Z]{3})(.*\n?)+\)$`
	if match := BodyTypePattern.FindStringSubmatch(messageBody); len(match) > 1 {
		name := match[1] // match[1] captures the 3-letter category code
		if bodyConfig, found := bodyPatterns[name]; found {
			return &bodyConfig
		}
	}
	return nil
}

// ParseBody attempts to parse a given message body string using the configurations found by FindPatterns.
// It iterates through the patterns defined in the matched BodyConfig.
// For the first pattern that successfully matches the messageBody, it extracts all named capture groups
// from the regex into a map[string]string, where keys are capture group names and values are the captured strings.
// messageBody: The raw string of the message body.
// Returns a map of extracted key-value pairs if a pattern matches, otherwise nil.
func ParseBody(messageBody string) map[string]string {
	if bodyConfig := FindPatterns(messageBody); bodyConfig != nil { // Renamed 'body' to 'bodyConfig' for clarity
		for _, pattern := range bodyConfig.Patterns {
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
