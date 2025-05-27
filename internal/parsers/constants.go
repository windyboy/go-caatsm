package parsers // Package comment already added via aviation_parser.go

import "regexp"

// String constants used throughout the parsers package.
const (
	// StartIndicatorPrefix is the standard prefix indicating the start of many aviation messages (e.g., SITA, AFTN).
	StartIndicatorPrefix = "ZCZC"
	// EndHeaderMarker is used in some message formats to explicitly denote the end of address/header fields.
	// Note: The value "." might be specific to a particular message type or interpretation.
	EndHeaderMarker = "."
	// BeginPartMarker indicates the start of a specific part within a multi-part message.
	BeginPartMarker = "BEGIN PART"

	// Category is a general key for the message category field.
	Category = "category"
	// CategoryArrival signifies an Arrival message (ARR).
	CategoryArrival = "ARR"
	// CategoryDeparture signifies a Departure message (DEP).
	CategoryDeparture = "DEP"
	// CategoryCancellation signifies a Cancellation message (CNL).
	CategoryCancellation = "CNL"
	// CategoryDelay signifies a Delay message (DLA).
	CategoryDelay = "DLA"
	// CategoryFlightPlan signifies a Filed Flight Plan message (FPL).
	CategoryFlightPlan = "FPL"
	// Note: CHG, CPL, PLN categories are defined in aviation_parser.go, ensure consistency if used here.

	// CANCELLED is a constant representing a cancelled status, often used with CNL messages.
	CANCELLED = "CNL" // This seems redundant with CategoryCancellation if used for the same purpose.
	// AirportCode is a general key for an airport code field.
	AirportCode = "airport"
	// Date is a general key for a date field.
	Date = "date"
	// Task is a key often used in schedule (PLN) messages for a task or category designator.
	Task = "task"
	// Index is a key for an index or sequence number.
	Index = "idx"
	// FlightNumber is a common key used for flight identification in message bodies.
	// This is different from the FlightNumber constant in aviation_parser.go which is "flight_number".
	// Consider standardizing key names.
	FlightNumber = "number"
	// Register is a key for aircraft registration.
	Register = "reg" // This clashes with the Register constant in aviation_parser.go ("reg"). Standardize.
)

// Regular expression patterns (as strings) used for parsing various message components.
// These are typically compiled into regexp.Regexp objects for use.
const (
	// AllDigitsPattern matches strings composed entirely of digits, capturing them as "dep_time".
	// The name "dep_time" suggests a specific use case, might need a more generic name if used broadly.
	AllDigitsPattern = `^(?P<dep_time>\d+)$`
	// IndexPattern matches various index or list marker formats.
	IndexPattern = `^(?P<idx>\(?L?[0-9]+\)?:?\.?)$`
	// DatePattern matches dates in DDMMM format (e.g., 30OCT).
	DatePattern = `^(?P<date>\d{2}\w{3})$`
	// TaskPattern matches task codes like "H/G".
	TaskPattern = `(?P<task>[A-Z]\/[A-Z])$`
	// WaypointPattern matches waypoint information, including optional arrival/departure times and airport codes.
	WaypointPattern = `^(SI:)?(?P<arr_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?\/?(?P<airport>[A-Z]{3})\/?(?P<dep_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?$`
	// FlightNumberPattern matches typical flight number formats.
	FlightNumberPattern = `^(?P<number>[0-9A-Z][0-9A-Z]\d{3,5}(\/\d+)*)$`
	// RegisterPattern matches aircraft registration formats starting with 'B' followed by 4 digits (e.g., B1234).
	RegisterPattern = `^(?P<reg>B\d{4})$`

	// ArrPatternString defines the regex for parsing Arrival (ARR) message bodies.
	ArrPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)(\/?(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})-(?P<arr>[A-Z]{4})(?P<arr_time>\d{4})\)$`
	// DepPatternString defines the regex for parsing Departure (DEP) message bodies.
	DepPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)(\/(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})-(?P<arr>[A-Z]{4})\)$`
	// FplPatternString defines the regex for parsing Flight Plan (FPL) message bodies. This is a complex, multi-line regex.
	FplPatternString = `\((?P<category>[A-Z]{3})-(?P<number>[A-Z]+\d+)-(?P<indicator>[A-Z]{2})\n-(?P<aircraft>[A-Z]+\d+\/?[A-Z]?)\n?-(?P<surve>.*)\n?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})\n?-(?P<speed>[A-Z]+\d+)(?P<level>[A-Z0-9]+)\s+(?P<route>(.|\n)+)\n-(?P<dest>[A-Z]{4})(?P<estt>\d{4})\s?(?P<alter>(\s[A-Z]{4})+)\n?-([A-Z]{3}\/(?:[A-Z]{4}\d{4}\s?)+)?(?P<other>(?m)[A-Z]{3}\/(.|\n)*)\)$`
	// CnlPatternString defines the regex for parsing Cancellation (CNL) message bodies.
	CnlPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})?-?(?<arr>[A-Z]{4})\)$`
	// DlaPatternString defines the regex for parsing Delay (DLA) message bodies.
	DlaPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})?-?(?<arr>[A-Z]{4})(?<arr_time>\d{4})?\)$`
)

// Compiled regular expressions used globally within the parsers package.
// These are pre-compiled for efficiency from the string constants defined above.
var (
	// AllDigitsExpression is the compiled version of AllDigitsPattern.
	AllDigitsExpression = regexp.MustCompile(AllDigitsPattern)
	// IndexExpression is the compiled version of IndexPattern.
	IndexExpression = regexp.MustCompile(IndexPattern)
	// TaskExpression is the compiled version of TaskPattern.
	TaskExpression = regexp.MustCompile(TaskPattern)
	// DateExpression is the compiled version of DatePattern.
	DateExpression = regexp.MustCompile(DatePattern)
	// WaypointExpression is the compiled version of WaypointPattern.
	WaypointExpression = regexp.MustCompile(WaypointPattern)
	// FlightNumberExpression is the compiled version of FlightNumberPattern.
	FlightNumberExpression = regexp.MustCompile(FlightNumberPattern)
	// RegisterExpression is the compiled version of RegisterPattern.
	RegisterExpression = regexp.MustCompile(RegisterPattern)

	// ArrPatternExpression is the compiled regex for ARR message bodies.
	ArrPatternExpression = regexp.MustCompile(ArrPatternString)
	// DepPatternExpression is the compiled regex for DEP message bodies.
	DepPatternExpression = regexp.MustCompile(DepPatternString)
	// FplPatternExpression is the compiled regex for FPL message bodies.
	FplPatternExpression = regexp.MustCompile(FplPatternString)
	// CnlPatternExpression is the compiled regex for CNL message bodies.
	CnlPatternExpression = regexp.MustCompile(CnlPatternString)
	// DlaPatternExpression is the compiled regex for DLA message bodies.
	DlaPatternExpression = regexp.MustCompile(DlaPatternString)

	// BodyTypePattern is a general regex to capture the message type from the start of a message body (e.g., "(ARR...").
	BodyTypePattern = regexp.MustCompile(`^\(([A-Z]{3})(.*\n?)+\)$`)

	// categoryRegex is used to find the message category within a message body, typically looking for "(CAT-" pattern.
	categoryRegex = regexp.MustCompile(`\((?P<category>[A-Z]+)-`)
	// emptyLineRemove is used to remove empty or whitespace-only lines from a message.
	emptyLineRemove = regexp.MustCompile(`(?m)^\s*$`)
	// bodyOnly attempts to extract the core message content, stripping potential SITA envelope data.
	bodyOnly = regexp.MustCompile(`(.|\n)?(ZCZC(.|\n)*)NNNN(.|\n)?$`)
	// originator is used to parse originator lines containing a timestamp and an originator ID.
	originator = regexp.MustCompile(`(?P<originatorDateTime>[0-9]+)\s(?P<originator>[A-Z]+)`)

	// Patterns for extracting data from FPL Field 18 (OtherInfo).
	// navPattern extracts NAV/ data.
	navPattern = regexp.MustCompile(`(?m)NAV\/(?P<nav>\w+)`)
	// remarkPattern extracts RMK/ data.
	remarkPattern = regexp.MustCompile(`(?s)RMK\/(?P<remark>.*)`)
	// selPattern extracts SEL/ (SELCAL) data.
	selPattern = regexp.MustCompile(`(?m)SEL\/(?P<sel>\w+)`)
	// regPattern extracts REG/ (Aircraft Registration) data.
	regPattern = regexp.MustCompile(`(?m)REG\/(?P<reg>[A-Z0-9]+)`)
	// pbnPattern extracts PBN/ (Performance-Based Navigation) data.
	pbnPattern = regexp.MustCompile(`(?m)PBN\/(?P<pbn>[A-Z0-9]+)`)
	// eetPattern extracts EET/ (Estimated Elapsed Time) data.
	eetPattern = regexp.MustCompile(`(?s)(-?EET\/(?P<eet>(?:[A-Z]{4}\d{4}\s*)+))`)
	// performancePattern extracts PER/ (Performance Category) data.
	performancePattern = regexp.MustCompile(`(?s)-?PER\/(?P<per>\w)`)
	// reroutePattern extracts RIF/ (Reroute Information) data.
	reroutePattern = regexp.MustCompile(`(?m)RIF\/(?P<rif>.*)[A-Z]{3}\/`)
)
