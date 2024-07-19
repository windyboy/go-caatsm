package parsers

import (
	"caatsm/internal/config"
	"caatsm/internal/domain"
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

type AFTNParser struct {
	BodyPattern *config.BodyConfig
}

// Parse parses a generic AFTN message based on its type.
func (p AFTNParser) Parse(text string) (interface{}, error) {
	data := ParseBody(text)

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
			AircraftID:       data["number"],
			SSRModeAndCode:   data["ssr"],
			DepartureAirport: data["departure"],
			DepartureTime:    data["departure_time"],
			Destination:      data["arrival"],
		}, nil
	default:
		return nil, fmt.Errorf("invalid message type")
	}
}

// removeEmptyLines removes empty lines from a given text.
func removeEmptyLines(text string) string {
	cleanedText := emptyLineRemove.ReplaceAllString(text, "")
	return strings.ReplaceAll(cleanedText, "\n\n", "\n")
}

// ParseAFTN parses an AFTN message from raw text.
func ParseAFTN(rawMessage string) (*domain.AFTN, error) {
	cleanedText := removeEmptyLines(rawMessage)
	lines := strings.Split(cleanedText, "\n")

	if len(lines) < 4 {
		return nil, fmt.Errorf("invalid AFTN message format")
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

	text, bodyType, err := parseText(strings.Join(lines[3:], "\n"))
	if err != nil {
		return nil, err
	}

	bodyData, err := parseBodyData(text)
	if err != nil {
		return nil, err
	}

	return &domain.AFTN{
		Header:            header,
		PriorityAndSender: priorityAndSender,
		TimeAndReceiver:   timeAndReceiver,
		Text:              text,
		Category:          bodyType,
		BodyData:          bodyData,
		ReceivedTime:      time.Now(),
	}, nil
}

// parseHeader parses the header line of an AFTN message.
func parseHeader(line string) (domain.Header, error) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return domain.Header{}, fmt.Errorf("invalid header format")
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
		return domain.PriorityAndSender{}, fmt.Errorf("invalid priority and sender format")
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
		return domain.TimeAndReceiver{}, fmt.Errorf("invalid time and receiver format")
	}
	return domain.TimeAndReceiver{
		Time:     parts[0],
		Receiver: parts[1],
	}, nil
}

// parseText parses the text and extracts the body type from an AFTN message.
func parseText(text string) (string, string, error) {
	match := textPattern.FindStringSubmatch(text)
	if len(match) > 1 {
		return match[0], match[1], nil
	}
	return "", "", fmt.Errorf("invalid text format")
}

// parseBodyData parses the body data of an AFTN message.
func parseBodyData(text string) (interface{}, error) {
	bodyParser := AFTNParser{}
	bodyData, err := bodyParser.Parse(text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body data: %w", err)
	}
	return bodyData, nil
}

// ValidateAFTN validates the fields of an AFTN message.
func ValidateAFTN(msg *domain.AFTN) error {
	if missingRequiredFields(msg) {
		return fmt.Errorf("invalid AFTN message: missing fields")
	}

	if !validPriority.MatchString(msg.PriorityAndSender.Priority) {
		return fmt.Errorf("invalid priority code")
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
