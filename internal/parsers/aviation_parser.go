package parsers

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/pkg/utils"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	StartIndicatorPrefix = "ZCZC"
	EndHeaderMarker      = "."
	BeginPartMarker      = "BEGIN PART"

	Category             = "category"
	CategoryArrival      = "ARR"
	FlightNumber         = "number"
	SSR                  = "ssr"
	DepartureCode        = "dep"
	DepartureTime        = "dep_time"
	ArrivalCode          = "arr"
	ArrivalTime          = "arr_time"
	CategoryDeparture    = "DEP"
	DestinationCode      = "dest"
	OtherInfo            = "other"
	CategoryCancellation = "CNL"
	CategoryDelay        = "DLA"
	NewDepartureTime     = "new_departure_time"
	CategoryFlightPlan   = "FPL"
	ReferenceData        = "reference_data"
	Aircraft             = "aircraft"
	CategorySurveillance = "surve"
	Indicator            = "indicator"
	Other                = "other"
	AircraftID           = "aircraft"
	Surveillance         = "surve"
	Speed                = "speed"
	Level                = "level"
	Route                = "route"
	EstimatedTime        = "estt"
	AlternateAirport     = "alter"
	Register             = "reg"
	PBN                  = "pbn"
	NavigationEquipment  = "nav"
	EstimatedElapsedTime = "eet"
	SELCALCode           = "sel"
	PerformanceCategory  = "per"
	RerouteInformation   = "rif"
	Remarks              = "remark"
	Index                = "idx"
)

var (
	categoryRegex      = regexp.MustCompile(`\((?P<category>[A-Z]+)-`)
	emptyLineRemove    = regexp.MustCompile(`(?m)^\s*$`)
	bodyOnly           = regexp.MustCompile(`(.|\n)?(ZCZC(.|\n)*)NNNN(.|\n)?$`)
	originator         = regexp.MustCompile(`(?P<originatorDateTime>[0-9]+)\s(?P<originator>[A-Z]+)`)
	navPattern         = regexp.MustCompile(`(?m)NAV\/(?P<nav>.*)$`)
	remarkPattern      = regexp.MustCompile(`(?s)RMK\/(?P<remark>.*)`)
	selPattern         = regexp.MustCompile(`(?m)SEL\/(?P<sel>\w+)`)
	pbnPattern         = regexp.MustCompile(`(?m)PBN\/(?P<pbn>[A-Z0-9]+)`)
	eetPattern         = regexp.MustCompile(`(?s)(-?EET\/(?P<eet>(?:[A-Z]{4}\d{4}\s*)+))`)
	performancePattern = regexp.MustCompile(`(?s)-?PER\/(?P<per>\w)`)
	reroutePattern     = regexp.MustCompile(`(?m)RIF\/(?P<rif>.*)[A-Z]{3}\/`)
	otherPatterns      = []*regexp.Regexp{navPattern, remarkPattern, selPattern, pbnPattern, eetPattern, performancePattern, reroutePattern}
)

type BodyParser struct {
	body         string
	bodyPatterns map[string]config.BodyConfig
	mu           sync.Mutex
}

func NewBodyParser(body string) *BodyParser {
	return &BodyParser{
		bodyPatterns: config.GetBodyPatterns(),
		body:         body,
	}
}

func (bp *BodyParser) GetBodyPatterns() map[string]config.BodyConfig {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.bodyPatterns
}

func (bp *BodyParser) SetBodyPatterns(patterns map[string]config.BodyConfig) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.bodyPatterns = patterns
}

func (bp *BodyParser) Parse() (string, interface{}, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.body = strings.TrimSpace(bp.body)
	category := findCategory(bp.body)
	if category == "" {
		return "", nil, fmt.Errorf("no category found in body text")
	}

	if patternConfig, exists := bp.bodyPatterns[category]; exists && patternConfig.Patterns != nil {
		for _, p := range patternConfig.Patterns {
			// if match := p.Expression.FindStringSubmatch(bp.body); match != nil {
			// 	data := extractData(match, p.Expression)
			// 	return bp.createBodyData(data)
			// }
			if data := parse(bp.body, p.Expression); data != nil {
				return bp.createBodyData(data)
			}
		}
	}
	return "", nil, fmt.Errorf("no matching pattern found for body: %s", bp.body)
}

func findCategory(body string) string {
	if match := categoryRegex.FindStringSubmatch(body); match != nil {
		for i, name := range categoryRegex.SubexpNames() {
			if i != 0 && name == "category" {
				return match[i]
			}
		}
	}
	return ""
}

func parse(data string, exp *regexp.Regexp) map[string]string {
	match := exp.FindStringSubmatch(data)
	if len(match) > 0 {
		return extractData(match, exp)
	}
	return nil
}

func extractData(match []string, re *regexp.Regexp) map[string]string {
	data := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			data[name] = strings.TrimSpace(match[i])
		}
	}
	return data
}

func (bp *BodyParser) createBodyData(data map[string]string) (string, interface{}, error) {
	switch category := data["category"]; category {
	case CategoryArrival:
		return category, &domain.ARR{
			Category:         data[Category],
			AircraftID:       data[FlightNumber],
			SSRModeAndCode:   data[SSR],
			DepartureAirport: data[DepartureCode],
			ArrivalAirport:   data[ArrivalCode],
			ArrivalTime:      data[ArrivalTime],
		}, nil
	case CategoryDeparture:
		return category, &domain.DEP{
			Category:         data[Category],
			AircraftID:       data[FlightNumber],
			SSRModeAndCode:   data[SSR],
			DepartureAirport: data[DepartureCode],
			DepartureTime:    data[DepartureTime],
			Destination:      data[ArrivalCode],
		}, nil
	case CategoryCancellation:
		return category, &domain.CNL{
			Category:           data[category],
			AircraftID:         data[FlightNumber],
			DepartureAirport:   data[DepartureCode],
			DestinationAirport: data[ArrivalCode],
		}, nil
	case CategoryDelay:
		return category, &domain.DLA{
			Category:         data[Category],
			AircraftID:       data[FlightNumber],
			DepartureAirport: data[DepartureCode],
			NewDepartureTime: data[DepartureTime],
			ArrivalAirport:   data[ArrivalCode],
			ArrivalTime:      data[ArrivalTime],
		}, nil
	case CategoryFlightPlan:
		otherData := parseOther(data[OtherInfo])
		return category, &domain.FPL{
			Category:                data[Category],
			FlightNumber:            data[FlightNumber],
			ReferenceData:           data[ReferenceData],
			AircraftID:              data[AircraftID],
			SSRModeAndCode:          data[Surveillance],
			FlightRulesAndType:      data[Indicator],
			CruisingSpeedAndLevel:   data[Speed] + data[Level],
			DepartureAirport:        data[DepartureCode],
			DepartureTime:           data[DepartureTime],
			Route:                   data[Route],
			DestinationAndTotalTime: data[DestinationCode] + data[EstimatedTime],
			AlternateAirport:        data[AlternateAirport],
			OtherInfo:               data[OtherInfo],
			Register:                otherData[Register],
			EstimatedArrivalTime:    data[EstimatedTime],
			PBN:                     otherData[PBN],
			NavigationEquipment:     otherData[NavigationEquipment],
			EstimatedElapsedTime:    otherData[EstimatedElapsedTime],
			SELCALCode:              otherData[SELCALCode],
			PerformanceCategory:     otherData[PerformanceCategory],
			RerouteInformation:      otherData[RerouteInformation],
			Remarks:                 otherData[Remarks],
		}, nil
	default:
		return category, nil, fmt.Errorf("invalid message type: %s", category)
	}
}

func Parse(rawText string) *domain.ParsedMessage {
	message, err := ParseHeader(rawText)
	if err != nil {
		msg := domain.NewParsedMessage()
		msg.Text = rawText
		return msg
	}

	bodyParser := NewBodyParser(message.Body)
	category, bodyData, err := bodyParser.Parse()
	message.Category = category
	message.ParsedAt = time.Now()

	if err != nil {
		message.Comments = err.Error()
		return &message
	}
	message.Parsed = true
	message.BodyData = bodyData
	return &message
}

func cleanMessage(text string) string {
	cleanedText := emptyLineRemove.ReplaceAllString(text, "")
	cleanText := strings.ReplaceAll(cleanedText, "\n\n", "\n")
	if match := bodyOnly.FindStringSubmatch(cleanText); len(match) > 1 {
		bodyContent := match[2]
		if bodyContent[len(bodyContent)-1] == '\n' {
			return bodyContent[:len(bodyContent)-1]
		}
		return bodyContent
	}
	return ""
}

func ParseHeader(fullMessage string) (domain.ParsedMessage, error) {
	log := utils.GetSugaredLogger()
	cleaned := cleanMessage(fullMessage)
	lines := strings.Split(cleaned, "\n")

	if len(lines) < 3 {
		log.Warnf("invalid message format: %s", fullMessage)
		return domain.ParsedMessage{Text: fullMessage}, fmt.Errorf("invalid message format: %s", fullMessage)
	}

	_, messageID, dateTime, err := parseStartIndicator(lines[0])
	if err != nil {
		return domain.ParsedMessage{Text: fullMessage}, err
	}

	priorityIndicator, primaryAddress := parsePriorityAndPrimary(lines[1])
	secondaryAddresses, originator, originatorDateTime, body := parseRemainingLines(lines[2:])

	return domain.ParsedMessage{
		MessageID:          messageID,
		DateTime:           dateTime,
		PriorityIndicator:  priorityIndicator,
		PrimaryAddress:     primaryAddress,
		SecondaryAddresses: secondaryAddresses,
		Originator:         originator,
		OriginatorDateTime: originatorDateTime,
		Text:               fullMessage,
		Body:               body,
		ReceivedAt:         time.Now(),
	}, nil
}

func parseStartIndicator(line string) (string, string, string, error) {
	parts := strings.Fields(line)
	if len(parts) >= 3 && strings.HasPrefix(parts[0], StartIndicatorPrefix) {
		return parts[0], parts[1], parts[2], nil
	}
	utils.GetSugaredLogger().Warnf("invalid start indicator line format: %s", line)
	return "", "", "", fmt.Errorf("invalid start indicator line format: %s", line)
}

func parsePriorityAndPrimary(line string) (string, string) {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	utils.GetSugaredLogger().Warnf("invalid priority and primary address line format: %s", line)
	return "", ""
}

func parseRemainingLines(lines []string) ([]string, string, string, string) {
	var (
		secondaryAddresses []string
		originator         string
		originatorDateTime string
		bodyAndFooter      strings.Builder
		headerEnded        bool
	)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if headerEnded {
			bodyAndFooter.WriteString(line + "\n")
		} else {
			switch {
			case line == EndHeaderMarker:
			case strings.HasPrefix(line, "."):
				originatorInfo := strings.Fields(line[1:])
				if len(originatorInfo) >= 2 {
					originator = originatorInfo[0]
					originatorDateTime = originatorInfo[1]
				}
				headerEnded = true
			case strings.HasPrefix(line, BeginPartMarker) || strings.HasPrefix(line, "("):
				headerEnded = true
				if strings.Index(line, "NNNN") > 0 {
					break
				}
				bodyAndFooter.WriteString(line + "\n")
			default:
				if o1, o2 := getOriginator(line); o1 != "" {
					originatorDateTime = o1
					originator = o2
				} else {
					secondaryAddresses = append(secondaryAddresses, line)
				}
			}
		}
	}

	return secondaryAddresses, originator, originatorDateTime, bodyAndFooter.String()
}

func getOriginator(line string) (string, string) {
	match := originator.FindStringSubmatch(line)
	if len(match) >= 3 {
		return match[1], match[2]
	}
	utils.GetSugaredLogger().Warnf("invalid originator line format: %s", line)
	return "", ""
}

func parseOther(text string) map[string]string {
	data := make(map[string]string)
	for _, re := range otherPatterns {
		if match := re.FindStringSubmatch(text); len(match) > 0 {
			for i, name := range re.SubexpNames() {
				if i != 0 && name != "" {
					data[name] = strings.TrimSpace(match[i])
				}
			}
		}
	}
	return data
}
