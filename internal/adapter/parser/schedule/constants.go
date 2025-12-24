package schedule

import "regexp"

// String constants
const (
	AirportCode  = "airport"
	Date         = "date"
	Task         = "task"
	Index        = "idx"
	FlightNumber = "number"
	Register     = "reg"
	ArrivalTime  = "arr_time"
	DepartureTime = "dep_time"
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
	cancelledPattern       = regexp.MustCompile(`\bCNL\b`)
)
