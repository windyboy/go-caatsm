package parsers

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	StartIndicatorPrefix = "ZCZC"
	EndHeaderMarker      = "."
	BeginPartMarker      = "BEGIN PART"
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
	bodyPatterns map[string]config.BodyConfig
}

// NewBodyParser initializes a BodyParser with the default body patterns.
func NewBodyParser() *BodyParser {
	return &BodyParser{bodyPatterns: config.GetBodyPatterns()}
}

// GetBodyPatterns returns the body patterns used by the parser.
func (bp *BodyParser) GetBodyPatterns() map[string]config.BodyConfig {
	return bp.bodyPatterns
}

// SetBodyPatterns sets the body patterns for the parser.
func (bp *BodyParser) SetBodyPatterns(patterns map[string]config.BodyConfig) {
	bp.bodyPatterns = patterns
}

// Parse attempts to parse the body text using the configured patterns.
func (bp *BodyParser) Parse(body string) (string, interface{}, error) {
	body = strings.TrimSpace(body)
	category := findCategory(body)
	if category == "" {
		return "", nil, fmt.Errorf("no category found in body text")
	}

	if patternConfig, exists := bp.bodyPatterns[category]; exists && patternConfig.Patterns != nil {
		for _, p := range patternConfig.Patterns {
			if match := p.Expression.FindStringSubmatch(body); match != nil {
				data := extractData(match, p.Expression)
				return createBodyData(data)
			}
		}
	}
	return "", nil, fmt.Errorf("no matching pattern found for body: %s", body)
}

// findCategory extracts the category from the body text using regex.
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

// extractData extracts named groups from the regex match and returns them as a map.
func extractData(match []string, re *regexp.Regexp) map[string]string {
	data := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			data[name] = match[i]
		}
	}
	return data
}

// createBodyData creates the appropriate domain object based on the type of message.
func createBodyData(data map[string]string) (string, interface{}, error) {
	switch category := data["category"]; category {
	case "ARR":
		return category, &domain.ARR{
			Category:         data["category"],
			AircraftID:       data["number"],
			SSRModeAndCode:   data["ssr"],
			DepartureAirport: data["departure"],
			ArrivalAirport:   data["arrival"],
			ArrivalTime:      data["time"],
		}, nil
	case "DEP":
		return category, &domain.DEP{
			Category:         data["category"],
			AircraftID:       data["number"],
			SSRModeAndCode:   data["ssr"],
			DepartureAirport: data["departure"],
			DepartureTime:    data["departure_time"],
			Destination:      data["arrival"],
		}, nil
	case "FPL":
		otherData := parseOther(data["other"])
		return category, &domain.FPL{
			Category:                data["category"],
			FlightNumber:            data["number"],
			ReferenceData:           data["reference_data"],
			AircraftID:              data["aircraft"],
			SSRModeAndCode:          data["surve"],
			FlightRulesAndType:      data["indicator"],
			CruisingSpeedAndLevel:   data["speed"] + data["level"],
			DepartureAirport:        data["departure"],
			DepartureTime:           data["departure_time"],
			Route:                   data["route"],
			DestinationAndTotalTime: data["destination"] + data["estt"],
			AlternateAirport:        data["alter"],
			OtherInfo:               data["other"],
			Register:                otherData["reg"],
			EstimatedArrivalTime:    data["estt"],
			PBN:                     otherData["pbn"],
			NavigationEquipment:     otherData["nav"],
			EstimatedElapsedTime:    otherData["eet"],
			SELCALCode:              otherData["sel"],
			PerformanceCategory:     otherData["per"],
			RerouteInformation:      otherData["rif"],
			Remarks:                 otherData["remark"],
		}, nil
	default:
		return category, nil, fmt.Errorf("cann't parse : %s", category)
	}
}

// Parse parses the raw text message and returns a ParsedMessage.
func Parse(rawText string) (*domain.ParsedMessage, error) {
	message, err := ParseHeader(rawText)
	if err != nil {
		return nil, err
	}

	bodyParser := NewBodyParser()
	category, bodyData, err := bodyParser.Parse(message.BodyAndFooter)
	message.Category = category
	message.ParsedAt = time.Now()

	if err != nil {
		return &message, err
	}

	message.BodyData = bodyData
	return &message, nil
}

// clean removes empty lines from a given text and extracts the body only.
func clean(text string) string {
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

// ParseHeader parses the header of the message and returns a ParsedMessage struct.
func ParseHeader(fullMessage string) (domain.ParsedMessage, error) {
	fullMessage = clean(fullMessage)
	lines := strings.Split(fullMessage, "\n")

	_, messageID, dateTime, err := parseStartIndicator(lines[0])
	if err != nil {
		return domain.ParsedMessage{}, err
	}

	priorityIndicator, primaryAddress := parsePriorityAndPrimary(lines[1])
	secondaryAddresses, originator, originatorDateTime, bodyAndFooter := parseRemainingLines(lines[2:])

	return domain.ParsedMessage{
		MessageID:          messageID,
		DateTime:           dateTime,
		PriorityIndicator:  priorityIndicator,
		PrimaryAddress:     primaryAddress,
		SecondaryAddresses: secondaryAddresses,
		Originator:         originator,
		OriginatorDateTime: originatorDateTime,
		BodyAndFooter:      bodyAndFooter,
		ReceivedAt:         time.Now(),
	}, nil
}

// parseStartIndicator parses the start indicator line.
func parseStartIndicator(line string) (string, string, string, error) {
	parts := strings.Fields(line)
	if len(parts) >= 3 && strings.HasPrefix(parts[0], StartIndicatorPrefix) {
		return parts[0], parts[1], parts[2], nil
	}
	return "", "", "", fmt.Errorf("invalid start indicator line format: %s", line)
}

// parsePriorityAndPrimary parses the priority indicator and primary address line.
func parsePriorityAndPrimary(line string) (string, string) {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// parseRemainingLines parses the remaining lines of the message.
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
				// End header marker, do nothing
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

// getOriginator extracts originator details from a line of text.
func getOriginator(line string) (string, string) {
	match := originator.FindStringSubmatch(line)
	if len(match) >= 3 {
		return match[1], match[2]
	}
	return "", ""
}

// parseOther parses additional information from the message body.
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
