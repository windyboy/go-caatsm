package parsers

import "regexp"

// String constants
const (
	StartIndicatorPrefix = "ZCZC"
	EndHeaderMarker      = "."
	BeginPartMarker      = "BEGIN PART"

	Category             = "category"
	CategoryArrival      = "ARR"
	CategoryDeparture    = "DEP"
	CategoryCancellation = "CNL"
	CategoryDelay        = "DLA"
	CategoryFlightPlan   = "FPL"

	CANCELLED    = "CNL"
	AirportCode  = "airport"
	Date         = "date"
	Task         = "task"
	Index        = "idx"
	FlightNumber = "number"
	Register     = "reg"
)

// Regular expression patterns
const (
	AllDigitsPattern    = `^(?P<dep_time>\d+)$`
	IndexPattern        = `^(?P<idx>\(?L?[0-9]+\)?:?\.?)$`
	DatePattern         = `^(?P<date>\d{2}\w{3})$`
	TaskPattern         = `(?P<task>[A-Z]\/[A-Z])$`
	WaypointPattern     = `^(SI:)?(?P<arr_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?\/?(?P<airport>[A-Z]{3})\/?(?P<dep_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?$`
	FlightNumberPattern = `^(?P<number>[0-9A-Z][0-9A-Z]\d{3,5}(\/\d+)*)$`
	RegisterPattern     = `^(?P<reg>B\d{4})$`

	ArrPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)(\/?(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})-(?P<arr>[A-Z]{4})(?P<arr_time>\d{4})\)$`
	DepPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>[A-Z0-9]+)(\/(?P<ssr>[A-Z0-9]+))?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})-(?P<arr>[A-Z]{4})\)$`
	FplPatternString = `\((?P<category>[A-Z]{3})-(?P<number>[A-Z]+\d+)-(?P<indicator>[A-Z]{2})\n-(?P<aircraft>[A-Z]+\d+\/?[A-Z]?)\n?-(?P<surve>.*)\n?-(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})\n?-(?P<speed>[A-Z]+\d+)(?P<level>[A-Z0-9]+)\s+(?P<route>(.|\n)+)\n-(?P<dest>[A-Z]{4})(?P<estt>\d{4})\s?(?P<alter>(\s[A-Z]{4})+)\n?-([A-Z]{3}\/(?:[A-Z]{4}\d{4}\s?)+)?(?P<other>(?m)[A-Z]{3}\/(.|\n)*)\)$`
	CnlPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})?-?(?<arr>[A-Z]{4})\)$`
	DlaPatternString = `^\((?P<category>[A-Z]{3})-(?P<number>\w+\d+)-?(?P<dep>[A-Z]{4})(?P<dep_time>\d{4})?-?(?<arr>[A-Z]{4})(?<arr_time>\d{4})?\)$`
)

// Compiled regular expressions
var (
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

	categoryRegex      = regexp.MustCompile(`\((?P<category>[A-Z]+)-`)
	emptyLineRemove    = regexp.MustCompile(`(?m)^\s*$`)
	bodyOnly           = regexp.MustCompile(`(.|\n)?(ZCZC(.|\n)*)NNNN(.|\n)?$`)
	originator         = regexp.MustCompile(`(?P<originatorDateTime>[0-9]+)\s(?P<originator>[A-Z]+)`)
	navPattern         = regexp.MustCompile(`(?m)NAV\/(?P<nav>\w+)`)
	remarkPattern      = regexp.MustCompile(`(?s)RMK\/(?P<remark>.*)`)
	selPattern         = regexp.MustCompile(`(?m)SEL\/(?P<sel>\w+)`)
	regPattern         = regexp.MustCompile(`(?m)REG\/(?P<reg>[A-Z0-9]+)`)
	pbnPattern         = regexp.MustCompile(`(?m)PBN\/(?P<pbn>[A-Z0-9]+)`)
	eetPattern         = regexp.MustCompile(`(?s)(-?EET\/(?P<eet>(?:[A-Z]{4}\d{4}\s*)+))`)
	performancePattern = regexp.MustCompile(`(?s)-?PER\/(?P<per>\w)`)
	reroutePattern     = regexp.MustCompile(`(?m)RIF\/(?P<rif>.*)[A-Z]{3}\/`)
)
