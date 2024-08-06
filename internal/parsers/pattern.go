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

// Constants for specific values.
const (
	CANCELLED   = "CNL"
	AirportCode = "airport"
	Date        = "date"
	Task        = "task"
	// Index       = "idx"
)

// Regular expression patterns.
const (
	AllDigitsPattern    = `^(?P<dep_time>\d+)$`
	IndexPattern        = `^(?P<idx>\(?L?[0-9]+\)?:?\.?)$`
	DatePattern         = `^(?P<date>\d{2}\w{3})$`
	TaskPattern         = `(?P<task>[A-Z]\/[A-Z])$`
	WaypointPattern     = `^(SI:)?(?P<arr_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?\/?(?P<airport>[A-Z]{3})\/?(?P<dep_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?$`
	FlightNumberPattern = `^(?P<number>[0-9A-Z][0-9A-Z]\d{3,5}(\/\d+)*)$`
	RegisterPattern     = `^(?P<reg>B\d{4})$`
	ArrPatternString    = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)(\/?(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})-(?P<arr>[A-Z]{4})(?P<arr_time>\d{4})\)$`
	DepPatternString    = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)(\/(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})-(?P<arr>[A-Z]{4})\)$`
	FplPatternString    = `\((?P<category>[A-Z]{3})-(?P<number>[A-Z]+\d+)-(?P<indicator>[A-Z]{2})\n-(?P<aircraft>[A-Z]+\d+\/?[A-Z]?)\n?-(?P<surve>.*)\n?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})\n?-(?P<speed>[A-Z]+\d+)(?P<level>[A-Z0-9]+)\s+(?P<route>(.|\n)+)\n-(?P<dest>[A-Z]{4})(?P<estt>\d{4})\s?(?P<alter>(\s[A-Z]{4})+)\n?-([A-Z]{3}\/(?:[A-Z]{4}\d{4}\s?)+)?(?P<other>(?m)[A-Z]{3}\/(.|\n)*)\)$`
	CnlPatternString    = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})?-?(?<arr>[A-Z]{4})\)$`
	DlaPatternString    = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})?-?(?<arr>[A-Z]{4})(?<arr_time>\d{4})?\)$`
)

var (
	// Pre-compiled regular expressions.
	AllDigitsExpression    = regexp.MustCompile(AllDigitsPattern)
	IndexExpression        = regexp.MustCompile(IndexPattern)
	TaskExpression         = regexp.MustCompile(TaskPattern)
	DateExpression         = regexp.MustCompile(DatePattern)
	WaypointExpression     = regexp.MustCompile(WaypointPattern)
	FlightNumberExpression = regexp.MustCompile(FlightNumberPattern)
	RegisterExpression     = regexp.MustCompile(RegisterPattern)
	ArrPatternExpression   = regexp.MustCompile(ArrPatternString)
	DepPatternExpression   = regexp.MustCompile(DepPatternString)
	FplPatternExpression   = regexp.MustCompile(FplPatternString)
	CnlPatternExpression   = regexp.MustCompile(CnlPatternString)
	DlaPatternExpression   = regexp.MustCompile(DlaPatternString)
	BodyTypePattern        = regexp.MustCompile(`^\(([A-Z]{3})(.*\n?)+\)$`)
	bodyPatterns           = map[string]BodyConfig{}
	parserMap              = map[string]*regexp.Regexp{}
	parserDef              = &[]LineParser{}
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
