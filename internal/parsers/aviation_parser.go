package parsers

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/pkg/utils"
	"errors" // Import errors package to handle errors
	"fmt"
	"strings"
)

const (
	StartIndicatorPrefix = "ZCZC"
	EndHeaderMarker      = "."
	BeginPartMarker      = "BEGIN PART"
)

type BodyParser struct {
	bodyPattern map[string]config.BodyConfig // Injected configuration for body patterns.
}

func DefaultBodyParser() *BodyParser {
	return &BodyParser{bodyPattern: config.GetBodyPatterns()}
}

func (bp *BodyParser) GetBodyPatterns() map[string]config.BodyConfig {
	return bp.bodyPattern
}

func (bp *BodyParser) SetBodyPatterns(patterns map[string]config.BodyConfig) {
	bp.bodyPattern = patterns
}

func (bp *BodyParser) Parse(body string) (interface{}, error) {
	log := utils.Logger
	data := make(map[string]string)
	for _, pattern := range bp.GetBodyPatterns() {
		for i, p := range pattern.Patterns {
			log.Debugf("Trying pattern %d: %s\n %s\n", i, p.Comments, p.Pattern)
			re := p.Expression
			match := re.FindStringSubmatch(body)
			if match != nil {
				log.Debugf("Matched : %v \n", match)
				for i, name := range re.SubexpNames() {
					if i != 0 && name != "" {
						data[name] = match[i]
					}
				}
				return CreateBodyData(data)
			}
			log.Debugf("No match for pattern %d\n", i)
		}
	}
	log.Errorf("No matching pattern found for body: %s", body)
	return data, errors.New("no matching pattern found for body")
}

func CreateBodyData(data map[string]string) (interface{}, error) {
	switch data["type"] {
	case "ARR":
		return &domain.ARR{
			Category:         data["type"],
			AircraftID:       data["number"],
			SSRModeAndCode:   data["ssr"],
			DepartureAirport: data["departure"],
			ArrivalAirport:   data["arrival"],
		}, nil
	case "DEP":
		return &domain.DEP{
			Category:         data["type"],
			AircraftID:       data["number"],
			SSRModeAndCode:   data["ssr"],
			DepartureAirport: data["departure"],
			DepartureTime:    data["departure_time"],
			Destination:      data["arrival"],
		}, nil
	case "FPL":
		return &domain.FPL{
			Category:                data["type"],
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
		return nil, fmt.Errorf("invalid message type: %s", data["type"])
	}
}

func Parse(rawText string) (*domain.ParsedMessage, error) {
	message, err := ParseHeader(rawText)
	if err != nil {
		return nil, err
	}
	bodyParser := DefaultBodyParser()
	bodyData, err := bodyParser.Parse(message.BodyAndFooter)
	if err != nil {
		return nil, err
	}
	message.BodyData = bodyData
	return &message, nil
}

// ParseHeader parses the header of the message and returns a ParsedMessage struct
func ParseHeader(fullMessage string) (domain.ParsedMessage, error) {
	fullMessage = strings.TrimSpace(fullMessage) // Trim leading and trailing spaces
	lines := strings.Split(fullMessage, "\n")    // Split the message into lines

	// Parse the start indicator line
	startIndicator, messageID, dateTime, err := parseStartIndicator(lines[0])
	if err != nil {
		return domain.ParsedMessage{}, err // Return error if parsing fails
	}

	// Parse the priority indicator and primary address line
	priorityIndicator, primaryAddress, err := parsePriorityAndPrimary(lines[1])
	if err != nil {
		return domain.ParsedMessage{}, err // Return error if parsing fails
	}

	// Parse the remaining lines
	secondaryAddresses, originator, originatorDateTime, bodyAndFooter := parseRemainingLines(lines[2:])

	// Return the parsed message as a domain.ParsedMessage struct
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
	}, nil
}

// parseStartIndicator parses the start indicator line
func parseStartIndicator(line string) (string, string, string, error) {
	if strings.HasPrefix(line, StartIndicatorPrefix) {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			return parts[0], parts[1], parts[2], nil
		}
	}
	return "", "", "", errors.New("invalid start indicator line format")
}

// parsePriorityAndPrimary parses the priority indicator and primary address line
func parsePriorityAndPrimary(line string) (string, string, error) {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}
	return "", "", errors.New("invalid priority indicator line format")
}

// parseRemainingLines parses the remaining lines of the message
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
			// Append line to body and footer if header has ended
			bodyAndFooter.WriteString(line + "\n")
		} else {
			switch {
			case line == EndHeaderMarker:
				// Ignore end header marker
			case strings.HasPrefix(line, "."):
				// Parse originator and originatorDateTime
				originatorInfo := strings.Fields(line[1:])
				if len(originatorInfo) >= 2 {
					originator = originatorInfo[0]
					originatorDateTime = originatorInfo[1]
				}
				headerEnded = true
			case strings.HasPrefix(line, BeginPartMarker) || strings.HasPrefix(line, "("):
				// Mark header as ended and append line to body and footer
				headerEnded = true
				bodyAndFooter.WriteString(line + "\n")
			default:
				// Append line to secondary addresses
				secondaryAddresses = append(secondaryAddresses, line)
			}
		}
	}

	return secondaryAddresses, originator, originatorDateTime, bodyAndFooter.String()
}
