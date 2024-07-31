package parsers

import (
	"caatsm/internal/domain"
	"caatsm/pkg/utils"
	"regexp"
	"strings"
)

const (
	AirportCode         = "airport"
	Date                = "date"
	Task                = "task"
	IndexPattern        = `^(?P<idx>\(?L?[0-9]+\)?\.?)$`
	DatePattern         = `^(?P<date>\d{2}\w{3})$`
	TaskPattern         = `^(?P<task>[A-Z]\/[A-Z])$`
	WaypointPattern     = `^(SI:)?(?P<arr_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?\/?(?P<airport>[A-Z]{3})\/?(?P<dep_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?$`
	FlightNumberPattern = `^(?P<number>[0-9A-Z][0-9A-Z]\d{3,5})$`
	RegisterPattern     = `^(?P<reg>B\d{4})$`
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

func ExtractWaypoint(message string) *domain.WayPoint {
	matches := WaypointExpression.FindStringSubmatch(message)
	if matches == nil {
		return nil
	}

	data := make(map[string]string)
	for i, name := range WaypointExpression.SubexpNames() {
		if i != 0 && name != "" {
			data[name] = matches[i]
		}
	}
	result := &domain.WayPoint{
		ArrivalTime:   data[ArrivalTime],
		Airport:       data[AirportCode],
		DepartureTime: data[DepartureTime],
	}

	return result
}

func ParseLine(line string) *domain.ScheduleLine {
	log := utils.GetSugaredLogger()
	cleanLine := strings.TrimSpace(line)
	words := strings.Split(cleanLine, " ")
	var flightSchedule = &domain.ScheduleLine{
		Reference: line,
	}
	var data map[string]string

	if indexData := parse(words[0], IndexExpression); indexData != nil {
		flightSchedule.Index = indexData[Index]
		words = words[1:]
	}

	// Define the parsing strategy
	parseStrategy := []string{
		Task,
		Date,
		FlightNumber,
		Register,
	}

	// Track parsed fields to avoid re-parsing
	parsed := make(map[string]bool)
	var maxParsed int

	// Parse each word in the line
	for i, word := range words {
		// Check if all fields have been parsed
		for _, name := range parseStrategy {
			// Skip if already parsed
			if parsed[name] {
				continue
			}
			// Parse the word
			if data = parse(word, parserMap[name]); data != nil {
				// Update the flight schedule
				switch name {
				case Task:
					flightSchedule.Task = data[Task]
					parsed[Task] = true
					maxParsed = i
				case Date:
					flightSchedule.Date = data[Date]
					parsed[Date] = true
					maxParsed = i
				case FlightNumber:
					//TODO: Handle multiple flight numbers
					flightSchedule.FlightNumber = append(flightSchedule.FlightNumber, data[FlightNumber])
					parsed[FlightNumber] = true
					maxParsed = i
				case Register:
					flightSchedule.AircraftReg = data[Register]
					parsed[Register] = true
					maxParsed = i
				}
				break
			}
		}

	}
	if maxParsed+1 < len(words) {
		flightSchedule.Waypoints = parseWaypoints(words[maxParsed+1:])
	} else {
		log.Warn("No waypoints found")
		flightSchedule.Comments = "No waypoints found"
	}

	return flightSchedule
}

func parseWaypoints(points []string) []domain.WayPoint {
	log := utils.GetSugaredLogger()
	if len(points) == 0 {
		log.Warn("No waypoints found")
		return nil
	}

	//find first waypoint
	var realWaypoints []string
	for i, point := range points {
		if parse(point, WaypointExpression) != nil {
			realWaypoints = points[i:]
			break
		}
	}
	if len(realWaypoints) == 0 {
		log.Warn("No waypoints found")
		return nil
	}
	var waypoints []domain.WayPoint
	for _, point := range realWaypoints {
		log.Debugf("Parsing waypoint: %s", point)
		if waypoint := ExtractWaypoint(point); waypoint != nil {
			log.Debugf("Waypoint: %v", waypoint)
			waypoints = append(waypoints, *waypoint)
		} else {
			log.Warnf("Failed to parse waypoint: %s", point)
		}
	}
	log.Debugf("Found %d waypoints", len(waypoints))
	return waypoints
}
