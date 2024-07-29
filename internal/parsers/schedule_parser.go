package parsers

import (
	"caatsm/internal/domain"
	"regexp"
	"strings"
)

const (
	// DepartureCode   = "dep"
	// DepartureTime   = "dep_time"
	// ArrivalCode     = "arr"
	AirportCode         = "airport"
	Date                = "date"
	Task                = "task"
	IndexPattern        = `(?P<idx>\(?L?\d+\)?\.?)`
	DatePattern         = `\s?(?P<date>\d{2}\w{3})`
	TaskPattern         = `\s?(?P<task>[A-Z]\/[A-Z])`
	WaypointPattern     = `\s?(?P<arr_time>\d{4}(\(\d{2}\w{3}\))?)\/?(?P<airport>\w{3})\/?(?P<dep_time>\d{4}(\(\d{2}\w{3}\))?)`
	FlightNumberPattern = `\s?(?P<number>[0-9A-Z][A-Z]\d{3,5})`
	RegisterPattern     = `\s?(?P<reg>B\d{4})`
)

var (
	IndexExpression        = regexp.MustCompile(IndexPattern)
	TaskExpression         = regexp.MustCompile(TaskPattern)
	DateExpression         = regexp.MustCompile(DatePattern)
	WaypointExpression     = regexp.MustCompile(WaypointPattern)
	FlightNumberExpression = regexp.MustCompile(FlightNumberPattern)
	RegisterExpression     = regexp.MustCompile(RegisterPattern)

	parserMap = map[string]*regexp.Regexp{
		Index:        IndexExpression,
		Task:         TaskExpression,
		Date:         DateExpression,
		FlightNumber: FlightNumberExpression,
		Register:     RegisterExpression,
	}
)

func FindWaypoints(message string) map[string]string {
	matches := WaypointExpression.FindStringSubmatch(message)
	if matches == nil {
		return nil
	}

	result := make(map[string]string)
	for i, name := range WaypointExpression.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = matches[i]
		}
	}

	return result
}

func ParseLine(line string) *domain.FlightSchedule {
	cleanLine := strings.TrimSpace(line)
	words := strings.Split(cleanLine, " ")
	var flightSchedule = &domain.FlightSchedule{}
	// var err error
	var data map[string]string

	// Define the parsing strategy
	parseStrategy := []string{
		Index,
		Task,
		Date,
		FlightNumber,
		Register,
	}

	// Track parsed fields to avoid re-parsing
	parsed := make(map[string]bool)

	for i, word := range words {
		if i > 0 {
			parsed[Index] = true
		}
		for _, name := range parseStrategy {
			if parsed[name] {
				continue
			}
			if data = parse(word, parserMap[name]); data != nil {

				switch name {
				case Index:
					flightSchedule.Index = data[Index]
				case Task:
					flightSchedule.Task = data[Task]
				case Date:
					flightSchedule.Date = data[Date]
				case FlightNumber:
					flightSchedule.FlightNumber = data[FlightNumber]
				case Register:
					flightSchedule.AircraftReg = data[Register]
				}
				parsed[name] = true
				break
			}
		}
	}

	return flightSchedule
}
