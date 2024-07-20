package parsers

import (
	"caatsm/internal/domain"
	"errors" // Import errors package to handle errors
	"strings"
)

const (
	StartIndicatorPrefix = "ZCZC"
	EndHeaderMarker      = "."
	BeginPartMarker      = "BEGIN PART"
)

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
