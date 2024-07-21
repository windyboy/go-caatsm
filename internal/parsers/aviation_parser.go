package parsers

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/pkg/utils"
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

var categoryRegex = regexp.MustCompile(`\(([A-Z]{3})(.*)\)`)
var emptyLineRemove = regexp.MustCompile(`(?m)^\s*$`)

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
func (bp *BodyParser) Parse(body string) (interface{}, error) {
	log := utils.Logger
	body = strings.TrimSpace(body)
	log.Info("Parsing body text", body)
	category := findCategory(body)
	if category == "" {
		log.Error("No category found in body text")
		return nil, fmt.Errorf("no category found in body text")
	}
	patters := bp.GetBodyPatterns()
	log.Infof("body config [%s] %v\n", category, patters[category])
	if patterConfig := patters[category]; patterConfig.Patterns != nil {

		for _, p := range patterConfig.Patterns {
			log.Infof("Trying pattern  %s\n%s\n", p.Comments, p.Pattern)

			re := p.Expression
			match := re.FindStringSubmatch(body)
			log.Info("Match: ", match)
			if match != nil {
				log.Infof("Matched: %v\n", match)
				data := extractData(match, re)
				return createBodyData(data)
			}
			log.Infof("No match for pattern %s\n", p.Comments)
		}

	}
	return nil, fmt.Errorf(" no matching pattern found for body: %s", body)
}

func findCategory(body string) string {
	match := categoryRegex.FindStringSubmatch(body)
	utils.Logger.Infof("Match: %v\n", match)
	if match != nil {
		return match[1]
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
func createBodyData(data map[string]string) (interface{}, error) {
	switch data["category"] {
	case "ARR":
		return &domain.ARR{
			Category:         data["category"],
			AircraftID:       data["number"],
			SSRModeAndCode:   data["ssr"],
			DepartureAirport: data["departure"],
			ArrivalAirport:   data["arrival"],
			ArrivalTime:      data["time"],
		}, nil
	case "DEP":
		return &domain.DEP{
			Category:         data["category"],
			AircraftID:       data["number"],
			SSRModeAndCode:   data["ssr"],
			DepartureAirport: data["departure"],
			DepartureTime:    data["departure_time"],
			Destination:      data["arrival"],
		}, nil
	case "FPL":
		return &domain.FPL{
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
			OtherInfo: fmt.Sprintf("%s %s REG/%s EET/%s SEL/%s PER/%s RIF/%s",
				data["pbn"], data["nav"], data["reg"], data["eet"], data["sel"], data["performance"], data["rif"]),
			SupplementaryInfo:    "RMK/" + data["remark"],
			EstimatedArrivalTime: data["estimated_arrival_time"],
			PBN:                  data["pbn"],
			NavigationEquipment:  data["nav"],
			EstimatedElapsedTime: data["eet"],
			SELCALCode:           data["sel"],
			PerformanceCategory:  data["performance"],
			RerouteInformation:   data["rif"],
			Remarks:              data["remark"],
		}, nil
	default:
		return nil, fmt.Errorf("invalid message type: %s", data["category"])
	}
}

// Parse parses the raw text message and returns a ParsedMessage.
func Parse(rawText string) (*domain.ParsedMessage, error) {
	message, err := ParseHeader(rawText)
	if err != nil {
		return nil, err
	}
	bodyParser := NewBodyParser()
	bodyData, err := bodyParser.Parse(message.BodyAndFooter)
	if err != nil {
		return nil, err
	}
	message.ParsedAt = time.Now()
	message.BodyData = bodyData
	return &message, nil
}

// removeEmptyLines removes empty lines from a given text.
func removeEmptyLines(text string) string {
	cleanedText := emptyLineRemove.ReplaceAllString(text, "")
	return strings.ReplaceAll(cleanedText, "\n\n", "\n")
}

// parseHeader parses the header of the message and returns a ParsedMessage struct.
func ParseHeader(fullMessage string) (domain.ParsedMessage, error) {
	fullMessage = removeEmptyLines(fullMessage)
	// fullMessage = strings.TrimSpace(fullMessage)
	lines := strings.Split(fullMessage, "\n")

	startIndicator, messageID, dateTime, err := parseStartIndicator(lines[0])
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
		StartIndicator:     startIndicator,
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
				secondaryAddresses = append(secondaryAddresses, line)
			}
		}
	}

	return secondaryAddresses, originator, originatorDateTime, bodyAndFooter.String()
}
