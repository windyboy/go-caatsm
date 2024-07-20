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

var (
	textPattern     = regexp.MustCompile(`\(([A-Z]{3})(.*)\)`)
	validPriority   = regexp.MustCompile(`^(SS|DD|FF|GG|KK)$`)
	validAddress    = regexp.MustCompile(`^[A-Z]{8}$`)
	emptyLineRemove = regexp.MustCompile(`(?m)^\s*$`)
)

// AFTNParser is responsible for parsing AFTN messages.
type AFTNParser struct {
	bodyPattern map[string]config.BodyConfig // Injected configuration for body patterns.
}

// NewAFTNParser creates a new instance of AFTNParser.
func NewAFTNParser(myPatterns map[string]config.BodyConfig) *AFTNParser {
	return &AFTNParser{bodyPattern: myPatterns}
}

// DefaultParser creates a new instance of AFTNParser with default patterns.
func DefaultParser() *AFTNParser {
	return &AFTNParser{bodyPattern: config.GetBodyPatterns()}
}

func (p *AFTNParser) GetBodyPatterns() map[string]config.BodyConfig {
	return p.bodyPattern
}

// Parse is the main method to parse an AFTN message.
func (p *AFTNParser) Parse(rawMessage string) (*domain.AFTN, error) {
	cleanedText := removeEmptyLines(rawMessage)
	lines := strings.Split(cleanedText, "\n")

	if len(lines) < 4 {
		return nil, fmt.Errorf("invalid AFTN message format: insufficient lines")
	}

	header, err := parseHeader(lines[0])
	if err != nil {
		return nil, err
	}

	priorityAndSender, err := parsePriorityAndSender(lines[1])
	if err != nil {
		return nil, err
	}

	timeAndReceiver, err := parseTimeAndReceiver(lines[2])
	if err != nil {
		return nil, err
	}

	body, category, err := extractBodyAndCategory(strings.Join(lines[3:], "\n"))
	if err != nil {
		return nil, err
	}

	bodyData, err := p.extractBodyData(body)
	if err != nil {
		return nil, err
	}

	aftn, err := p.createAFTN(bodyData)
	if err != nil {
		return nil, err
	}

	return &domain.AFTN{
		Header:            header,
		PriorityAndSender: priorityAndSender,
		TimeAndReceiver:   timeAndReceiver,
		Body:              body,
		Category:          category,
		BodyData:          aftn,
		ReceivedTime:      time.Now(),
	}, nil
}

// parseHeader parses the header line of an AFTN message.
func parseHeader(line string) (domain.Header, error) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return domain.Header{}, fmt.Errorf("invalid header format: %s", line)
	}
	return domain.Header{
		StartSignal: parts[0],
		SendID:      parts[1],
		SendTime:    parts[2],
	}, nil
}

// parsePriorityAndSender parses the priority and sender line of an AFTN message.
func parsePriorityAndSender(line string) (domain.PriorityAndSender, error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return domain.PriorityAndSender{}, fmt.Errorf("invalid priority and sender format: %s", line)
	}
	return domain.PriorityAndSender{
		Priority: parts[0],
		Sender:   parts[1],
	}, nil
}

// parseTimeAndReceiver parses the time and receiver line of an AFTN message.
func parseTimeAndReceiver(line string) (domain.TimeAndReceiver, error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return domain.TimeAndReceiver{}, fmt.Errorf("invalid time and receiver format: %s", line)
	}
	return domain.TimeAndReceiver{
		Time:     parts[0],
		Receiver: parts[1],
	}, nil
}

// parseTextInfo parses the text and extracts the body type from an AFTN message.
func extractBodyAndCategory(text string) (string, string, error) {
	match := textPattern.FindStringSubmatch(text)
	if len(match) > 1 {
		return match[0], match[1], nil
	}
	return "", "", fmt.Errorf("invalid text format: %s", text)
}

func (p *AFTNParser) ParseBody(body string) (interface{}, error) {
	bodyData, err := p.extractBodyData(body)
	if err != nil {
		return nil, err
	}

	aftn, err := p.createAFTN(bodyData)
	if err != nil {
		return nil, err
	}

	return aftn, nil
}

// parseBody parses the body data of an AFTN message.
func (p *AFTNParser) createAFTN(data map[string]string) (interface{}, error) {
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

// extractBodyData extracts the body data from an AFTN message.
func (p *AFTNParser) extractBodyData(text string) (map[string]string, error) {
	log := utils.Logger
	data := make(map[string]string)
	for _, pattern := range p.GetBodyPatterns() {
		for i, p := range pattern.Patterns {
			log.Debugf("Trying pattern %d: %s\n %s\n", i, p.Comments, p.Pattern)
			re := p.Expression
			match := re.FindStringSubmatch(text)
			if match != nil {
				log.Debugf("Matched : %v \n", match)
				for i, name := range re.SubexpNames() {
					if i != 0 && name != "" {
						data[name] = match[i]
					}
				}
				return data, nil
			}
			log.Debugf("No match for pattern %d\n", i)
		}
	}
	log.Errorf("No matching pattern found for text: %s", text)
	return nil, fmt.Errorf("no matching pattern found for text: %s", text)
}

// removeEmptyLines removes empty lines from a given text.
func removeEmptyLines(text string) string {
	return emptyLineRemove.ReplaceAllString(text, "")
}

// ValidateAFTN validates the fields of an AFTN message.
func ValidateAFTN(msg *domain.AFTN) error {
	if missingRequiredFields(msg) {
		return fmt.Errorf("invalid AFTN message: missing fields")
	}

	if !validPriority.MatchString(msg.PriorityAndSender.Priority) {
		return fmt.Errorf("invalid priority code: %s", msg.PriorityAndSender.Priority)
	}

	if !validAddress.MatchString(msg.TimeAndReceiver.Receiver) || !validAddress.MatchString(msg.PriorityAndSender.Sender) {
		return fmt.Errorf("invalid address format")
	}

	return nil
}

// missingRequiredFields checks if required fields in an AFTN message are missing.
func missingRequiredFields(msg *domain.AFTN) bool {
	return msg.PriorityAndSender.Priority == "" ||
		msg.TimeAndReceiver.Receiver == "" ||
		msg.PriorityAndSender.Sender == "" ||
		msg.Header.StartSignal == "" ||
		msg.Header.SendID == "" ||
		msg.Header.SendTime == ""
}
