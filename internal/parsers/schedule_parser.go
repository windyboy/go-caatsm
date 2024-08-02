package parsers

import (
	"caatsm/internal/domain"
	"caatsm/pkg/utils"

	"strings"
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

func FindDef(code string) *LineParser {
	// fmt.Printf("Finding definition for %s\n", code)
	// fmt.Println("ParserDef: ", parserDef)
	for _, def := range *parserDef {
		for _, airline := range def.Airlines {
			if airline == code {
				return &def
			}
		}
	}
	return nil
}

func standardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func ParseWithDef(line string, parserDef *LineParser) *domain.ScheduleLine {
	log := utils.GetSugaredLogger()
	cleanLine := standardizeSpaces(strings.TrimSpace(line))
	words := strings.Split(cleanLine, " ")
	var flightSchedule = &domain.ScheduleLine{
		Reference: line,
	}
	if strings.Contains(line, CANCELLED) {
		flightSchedule.Comments = "Cancelled"
		return flightSchedule
	}
	// var result map[string]string
	if parserDef == nil {
		log.Warnf("No definition found: %s", line)
		flightSchedule.Comments = "No definition found [" + line + "] "
		return flightSchedule
	}
	if len(words) < parserDef.MinLen {
		log.Warnf("Line too short: %s", line)
		flightSchedule.Comments = "Line too short [" + line + "] "
		return flightSchedule
	}

	for i, field := range parserDef.Fields {
		// log.Debugf("Parsing field %v -> %s", i, field)
		data := extract(words[i], parserMap[field])
		if data != nil {
			switch field {
			case Index:
				flightSchedule.Index = data[Index]
			case Date:
				flightSchedule.Date = data[Date]
			case Task:
				flightSchedule.Task = data[Task]
			case FlightNumber:
				flightSchedule.FlightNumber = append(flightSchedule.FlightNumber, data[FlightNumber])
			case Register:
				flightSchedule.AircraftReg = data[Register]
			}
		}
	}
	if len(words) > parserDef.WaypointStart {
		flightSchedule.Waypoints = parseWaypoints(words[parserDef.WaypointStart:])
	} else {
		log.Warn("No waypoints found")
		flightSchedule.Comments = "No waypoints found"
	}

	return flightSchedule
}

func ParseLine(line string) *domain.ScheduleLine {
	log := utils.GetSugaredLogger()
	cleanLine := strings.TrimSpace(line)
	words := strings.Split(cleanLine, " ")
	var flightSchedule = &domain.ScheduleLine{
		Reference: line,
	}
	var data map[string]string

	if indexData := extract(words[0], IndexExpression); indexData != nil {
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
			if data = extract(word, parserMap[name]); data != nil {
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
		if extract(point, WaypointExpression) != nil {
			realWaypoints = points[i:]
			break
		}
	}
	if len(realWaypoints) == 0 {
		log.Warn("No waypoints found")
		return nil
	}
	var waypoints []domain.WayPoint
	for i, point := range realWaypoints {
		// log.Debugf("Parsing waypoint: %s", point)
		// check the next point for departure time
		if digits := extract(point, AllDigitsExpression); i > 0 && digits != nil {
			// if the previous point was a waypoint, update the departure time
			if l := len(waypoints); l > 0 {
				waypoints[l-1].DepartureTime = digits[DepartureTime]
			}
		} else if waypoint := ExtractWaypoint(point); waypoint != nil {
			// log.Debugf("Waypoint: %v", waypoint)
			waypoints = append(waypoints, *waypoint)
		} else {
			log.Warnf("Failed to parse waypoint: %s", point)
		}
	}
	// log.Debugf("Found %d waypoints", len(waypoints))
	return waypoints
}
