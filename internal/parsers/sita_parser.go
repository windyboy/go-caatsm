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
	sitaTextPattern   = regexp.MustCompile(`\(([A-Z]{3})(.*)\)`)
	validSitaPriority = regexp.MustCompile(`^(SS|DD|FF|GG|KK)$`)
	validSitaAddress  = regexp.MustCompile(`^[A-Z]{8}$`)
)

// SITAParser is responsible for parsing SITA messages.
type SITAParser struct {
	bodyPattern map[string]config.BodyConfig // Injected configuration for body patterns.
}

// NewSITAParser creates a new instance of SITAParser.
func NewSITAParser(myPatterns map[string]config.BodyConfig) *SITAParser {
	return &SITAParser{bodyPattern: myPatterns}
}

// DefaultSITAParser creates a new instance of SITAParser with default patterns.
func DefaultSITAParser() *SITAParser {
	return &SITAParser{bodyPattern: config.GetBodyPatterns()}
}

func (p *SITAParser) GetBodyPatterns() map[string]config.BodyConfig {
	return p.bodyPattern
}

// Parse is the main method to parse a SITA message.
func (p *SITAParser) Parse(rawMessage string) (*domain.SITA, error) {
	cleanedText := removeEmptyLines(rawMessage)
	lines := strings.Split(cleanedText, "\n")

	if len(lines) < 4 {
		return nil, fmt.Errorf("invalid SITA message format: insufficient lines")
	}

	header, err := parseSITAHeader(lines[0])
	if err != nil {
		return nil, err
	}

	priorityAndSender, err := parseSITAPriorityAndSender(lines[1])
	if err != nil {
		return nil, err
	}

	timeAndReceiver, err := parseSITATimeAndReceiver(lines[2])
	if err != nil {
		return nil, err
	}

	text, bodyType, err := parseSITATextInfo(strings.Join(lines[3:], "\n"))
	if err != nil {
		return nil, err
	}

	bodyData, err := p.extractBodyData(text)
	if err != nil {
		return nil, err
	}

	sita, err := p.createSITA(bodyData)
	if err != nil {
		return nil, err
	}

	return &domain.SITA{
		Header:            header,
		PriorityAndSender: priorityAndSender,
		TimeAndReceiver:   timeAndReceiver,
		Text:              text,
		Category:          bodyType,
		BodyData:          sita,
		ReceivedTime:      time.Now(),
	}, nil
}

// parseSITAHeader parses the header line of a SITA message.
func parseSITAHeader(line string) (domain.SITAHeader, error) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return domain.SITAHeader{}, fmt.Errorf("invalid header format: %s", line)
	}
	return domain.SITAHeader{
		StartSignal: parts[0],
		SendID:      parts[1],
		SendTime:    parts[2],
	}, nil
}

// parseSITAPriorityAndSender parses the priority and sender line of a SITA message.
func parseSITAPriorityAndSender(line string) (domain.PrioritySender, error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return domain.PrioritySender{}, fmt.Errorf("invalid priority and sender format: %s", line)
	}
	return domain.PrioritySender{
		Priority: parts[0],
		Sender:   parts[1],
	}, nil
}

// parseSITATimeAndReceiver parses the time and receiver line of a SITA message.
func parseSITATimeAndReceiver(line string) (domain.TimeReceiver, error) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return domain.TimeReceiver{}, fmt.Errorf("invalid time and receiver format: %s", line)
	}
	return domain.TimeReceiver{
		Time:     parts[0],
		Receiver: parts[1],
	}, nil
}

// parseSITATextInfo parses the text and extracts the body type from a SITA message.
func parseSITATextInfo(text string) (string, string, error) {
	match := sitaTextPattern.FindStringSubmatch(text)
	if len(match) > 1 {
		return match[0], match[1], nil
	}
	return "", "", fmt.Errorf("invalid text format: %s", text)
}

func (p *SITAParser) ParseBody(body string) (interface{}, error) {
	bodyData, err := p.extractBodyData(body)
	if err != nil {
		return nil, err
	}

	sita, err := p.createSITA(bodyData)
	if err != nil {
		return nil, err
	}

	return sita, nil
}

// createSITA parses the body data of a SITA message.
func (p *SITAParser) createSITA(data map[string]string) (interface{}, error) {
	switch data["type"] {

	default:
		return nil, fmt.Errorf("invalid message type: %s", data["type"])
	}
}

// extractBodyData extracts the body data from a SITA message.
func (p *SITAParser) extractBodyData(text string) (map[string]string, error) {
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

// ValidateSITA validates the fields of a SITA message.
func ValidateSITA(msg *domain.SITA) error {
	if missingSitaFields(msg) {
		return fmt.Errorf("invalid SITA message: missing fields")
	}

	if !validSitaPriority.MatchString(msg.PriorityAndSender.Priority) {
		return fmt.Errorf("invalid priority code: %s", msg.PriorityAndSender.Priority)
	}

	if !validSitaAddress.MatchString(msg.TimeAndReceiver.Receiver) || !validSitaAddress.MatchString(msg.PriorityAndSender.Sender) {
		return fmt.Errorf("invalid address format")
	}

	return nil
}

func missingSitaFields(msg *domain.SITA) bool {
	if msg.Header.StartSignal == "" || msg.Header.SendID == "" || msg.Header.SendTime == "" ||
		msg.PriorityAndSender.Priority == "" || msg.PriorityAndSender.Sender == "" ||
		msg.TimeAndReceiver.Time == "" || msg.TimeAndReceiver.Receiver == "" ||
		msg.Category == "" {
		return true
	}
	return false
}
