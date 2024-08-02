package parsers

import (
	"caatsm/internal/domain"
	"caatsm/pkg/utils"
	"regexp"
	"strings"
)

type LineParser struct {
	Airlines      []string
	MinLen        int
	WaypointStart int
	Fields        map[int]string
}

const (
	CANCELLED   = "CNL"
	AirportCode = "airport"
	Date        = "date"
	Task        = "task"
	// Index               = "idx"
	AllDigitsPattern    = `^(?P<dep_time>\d+)$`
	IndexPattern        = `^(?P<idx>\(?L?[0-9]+\)?:?\.?)$`
	DatePattern         = `^(?P<date>\d{2}\w{3})$`
	TaskPattern         = `^(?P<task>[A-Z]\/[A-Z])$`
	WaypointPattern     = `^(SI:)?(?P<arr_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?\/?(?P<airport>[A-Z]{3})\/?(?P<dep_time>\d{4}(\(\d{2}[A-Z]{3}\))?)?$`
	FlightNumberPattern = `^(?P<number>[0-9A-Z][0-9A-Z]\d{3,5})$`
	RegisterPattern     = `^(?P<reg>B\d{4})$`
)

var (
	AllDigitsExpression = regexp.MustCompile(AllDigitsPattern)

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

	parserDef = &[]LineParser{
		{
			/**
			* 上航（FM）解析
			* 解析参考
			* W/Z FM9134 B2688 1/1ILS (00) TSN0100 SHA
			* W/Z FM9133 B2688 1/1ILS (00) SHA0340 TSN
			 */
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
			/**
			* 解析厦航(MF)计划
			* 01) MF8193 B5595 ILS(8) HGH1100 1305TSN
			* 02) MF8194 B5595 ILS(8) TSN1355 1550HGH
			 */
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
			/**
			* 解析奥凯(8X)计划
			* 计划参考：
			*L1:  29OCT  BK2735 B2863  ILS  IS (3/6)  TSN2350(28OCT)   HAK
			*L2:  29OCT  BK2735 B2863  ILS  IS (3/6)  HAK0435   NKG
			 */
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
			/**
			* 解析海航(HU)计划
			* 计划参考
			*L04 W/Z HU7204 B5637 (9) SZX/0500 TSN
			*L05 W/Z HU7205 B5406 (9) TSN/2355(30OCT) PVG
			*1)  JD5195 B6727 ILS I(9) SYX/0800 1135/TSN
			*
			*L07 W/Z GS6571 B3155 (7) XIY/0025 TSN/0245 CGQ
			 */
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
			/**
			* 1)  JD5195 B6727 ILS I(9) SYX/0800 1135/TSN
			 */
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
			/**
			* 01 GS7635 B3193 XIY0020(16APR) CGD
			 */
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
			/**
			* 杨子快运(Y8)
			*01 Y87969 B2119 XMN 1540 HGH
			*13 Y87444 B2578 ICN 0235 TSN
			 */
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
			/**
			* 四川航空 (3U)
			* 01)  31OCT 3U8863 B6598 CAT1 (10) CKG0010 0235TSN
			 */
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
