package parsers

import (
	"caatsm/internal/domain"
	"strings"
)

// ParseHeader parses the header of the message and returns a ParsedMessage struct
func ParseHeader(fullMessage string) domain.ParsedMessage {
	var (
		startIndicator     string
		messageID          string
		dateTime           string
		priorityIndicator  string
		primaryAddress     string
		secondaryAddresses []string
		originator         string
		originatorDateTime string
		bodyAndFooter      strings.Builder
		headerEnded        bool
	)

	fullMessage = strings.TrimSpace(fullMessage)
	lines := strings.Split(fullMessage, "\n")

	for lineCounter, line := range lines {
		switch lineCounter {
		case 0:
			if strings.HasPrefix(line, "ZCZC") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					startIndicator = parts[0]
					messageID = parts[1]
					dateTime = parts[2]
				}
			}
		case 1:
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				priorityIndicator = parts[0]
				primaryAddress = parts[1]
			}
		default:
			if !headerEnded {
				if strings.TrimSpace(line) == "." {
					continue
				} else if strings.HasPrefix(line, ".") {
					originatorInfo := strings.Fields(line[1:])
					if len(originatorInfo) >= 2 {
						originator = originatorInfo[0]
						originatorDateTime = originatorInfo[1]
					}
					headerEnded = true
				} else if strings.HasPrefix(line, "BEGIN PART") || strings.HasPrefix(line, "(") {
					headerEnded = true
					bodyAndFooter.WriteString(line + "\n")
				} else {
					secondaryAddresses = append(secondaryAddresses, line)
				}
			} else {
				bodyAndFooter.WriteString(line + "\n")
			}
		}
	}

	return domain.ParsedMessage{
		StartIndicator:     startIndicator,
		MessageID:          messageID,
		DateTime:           dateTime,
		PriorityIndicator:  priorityIndicator,
		PrimaryAddress:     primaryAddress,
		SecondaryAddresses: secondaryAddresses,
		Originator:         originator,
		OriginatorDateTime: originatorDateTime,
		BodyAndFooter:      bodyAndFooter.String(),
	}
}
