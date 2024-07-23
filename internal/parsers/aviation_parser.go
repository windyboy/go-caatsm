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
	categoryRegex   = regexp.MustCompile(`\((?P<category>[A-Z]+)-`)
	emptyLineRemove = regexp.MustCompile(`(?m)^\s*$`)
	bodyOnly        = regexp.MustCompile(`(.|\n)?(ZCZC(.|\n)*)NNNN(.|\n)?$`)
	originator      = regexp.MustCompile(`(?P<originatorDateTime>[0-9]+)\s(?P<originator>[A-Z]+)`)
	navPattern      = regexp.MustCompile(`(?m)^NAV\/(?P<nav>.*)$`)
	remarkPattern   = regexp.MustCompile(`(?s)^RMK\/(?P<remark>.*)$`)
	selPattern      = regexp.MustCompile(`(?m)SEL\/(?P<sel>\w+)`)
	pbnPattern      = regexp.MustCompile(`(?m)PBN\/(?P<pbn>[A-Z0-9]+)`)
	otherPatterns   = []regexp.Regexp{*navPattern, *remarkPattern, *selPattern, *pbnPattern}
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
	// log := utils.Logger
	body = strings.TrimSpace(body)
	// log.Info("Parsing body text", body)
	category := findCategory(body)
	if category == "" {
		// log.Error("No category found in body text")
		return "", nil, fmt.Errorf("no category found in body text")
	}
	patters := bp.GetBodyPatterns()
	// log.Infof("body config [%s] %v\n", category, patters[category])
	if patterConfig := patters[category]; patterConfig.Patterns != nil {

		for _, p := range patterConfig.Patterns {
			// log.Infof("Trying pattern  %s\n%s\n", p.Comments, p.Pattern)

			re := p.Expression
			match := re.FindStringSubmatch(body)
			// log.Info("Match: ", match)
			if match != nil {
				// log.Infof("Matched: %v\n", match)
				data := extractData(match, re)

				return createBodyData(data)
			}
			// log.Infof("No match for pattern %s\n", p.Comments)
		}

	}
	return "", nil, fmt.Errorf(" no matching pattern found for body: %s", body)
}

func findCategory(body string) string {
	match := categoryRegex.FindStringSubmatch(body)
	// utils.Logger.Infof("Match: %v\n", match)
	if match != nil {
		groups := categoryRegex.SubexpNames()
		for i, name := range groups {
			if i != 0 && name == "category" {
				return match[i]
			}
		}
	}
	return ""
}

// extractData extracts named groups from the regex match.
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
	category := data["category"]
	switch data["category"] {
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
			EstimatedElapsedTime:    data["eet"],
			SELCALCode:              otherData["sel"],
			// PerformanceCategory:     data["performance"],
			// RerouteInformation:      data["rif"],
			Remarks: otherData["remark"],
		}, nil
	default:
		return category, nil, fmt.Errorf("invalid message type: %s", category)
	}
}

// Parse parses the raw text message and returns a ParsedMessage.
// Parse parses the raw text message and returns a ParsedMessage.
func Parse(rawText string) (*domain.ParsedMessage, error) {
	// Parse the header of the message
	message, err := ParseHeader(rawText)
	if err != nil {
		return nil, err
	}

	// Initialize a new body parser
	bodyParser := NewBodyParser()

	// Parse the body and footer of the message
	category, bodyData, err := bodyParser.Parse(message.BodyAndFooter)

	message.Category = category

	if err != nil {
		// Return the message with the parsed header and the error
		message.ParsedAt = time.Now()
		return &message, err
	}

	// Set the parsed time to the current time
	message.ParsedAt = time.Now()

	// Assign the parsed body data to the message
	message.BodyData = bodyData

	// Return the fully parsed message
	return &message, nil
}

// removeEmptyLines removes empty lines from a given text.
func clean(text string) string {
	cleanedText := emptyLineRemove.ReplaceAllString(text, "")
	cleanText := strings.ReplaceAll(cleanedText, "\n\n", "\n")
	// if bodyOnly != nil {
	match := bodyOnly.FindStringSubmatch(cleanText)
	if len(match) > 1 {
		bodyOnly := match[2]
		if bodyOnly[len(bodyOnly)-1] == '\n' {
			return bodyOnly[:len(bodyOnly)-1]
		}
		return bodyOnly
	}
	// }
	return ""
}

// parseHeader parses the header of the message and returns a ParsedMessage struct.
func ParseHeader(fullMessage string) (domain.ParsedMessage, error) {
	fullMessage = clean(fullMessage)
	// fullMessage = strings.TrimSpace(fullMessage)
	lines := strings.Split(fullMessage, "\n")

	_, messageID, dateTime, err := parseStartIndicator(lines[0])
	if err != nil {
		return domain.ParsedMessage{}, err
	}

	// priorityIndicator, primaryAddress, err := parsePriorityAndPrimary(lines[1])
	// if err != nil {
	// 	return domain.ParsedMessage{}, err
	// }
	priorityIndicator, primaryAddress := parsePriorityAndPrimary(lines[1])

	secondaryAddresses, originator, originatorDateTime, bodyAndFooter := parseRemainingLines(lines[2:])

	return domain.ParsedMessage{
		// StartIndicator:     startIndicator,
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

func getOriginator(line string) (string, string) {
	match := originator.FindStringSubmatch(line)
	if len(match) >= 3 {
		return match[1], match[2]
	}
	return "", ""

}

func parseOther(text string) map[string]string {
	// fmt.Printf("Parsing other: %s\n", text)
	data := make(map[string]string)
	for _, re := range otherPatterns {
		match := re.FindStringSubmatch(text)
		if len(match) > 0 { // Corrected condition
			// fmt.Println("Matched: ", match)
			for i, name := range re.SubexpNames() {
				// fmt.Printf("index: %d, name: %s\n", i, name)
				if i != 0 && name != "" {
					data[name] = match[i]
				}
			}
		}
	}
	return data // Return the data map instead of nil
}
