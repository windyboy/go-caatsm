package domain // Already has package comment

import (
	"caatsm/pkg/utils"
	"fmt"
	"time"
)

// SITA represents a generic SITA message structure.
// It encapsulates header information, priority, sender/receiver details,
// the main text content, and metadata about its processing.
type SITA struct {
	Header            SITAHeader     `json:"header"`          // Header information of the SITA telegram.
	PriorityAndSender PrioritySender `json:"priority_sender"` // PriorityAndSender contains priority and sender address details.
	TimeAndReceiver   TimeReceiver   `json:"time_receiver"`   // TimeAndReceiver contains time and receiver address details.
	Text              string         `json:"text"`            // Text is the main content/body of the SITA telegram.
	ReceivedTime      time.Time      `json:"received_time"`   // ReceivedTime is the timestamp when the SITA message was received by the system.
	Category          string         `json:"category"`        // Category is an application-specific category assigned to the message (e.g., based on content).
	BodyData          interface{}    `json:"body_data"`       // BodyData can hold structured data parsed from the Text field.
}

// SITAHeader defines the standard header part of a SITA telegram.
type SITAHeader struct {
	StartSignal string `json:"start_signal"` // StartSignal indicates the beginning of the telegram (e.g., "ZCZC").
	SendID      string `json:"send_id"`      // SendID is a sending identifier, often a sequence number or unique ID from the source.
	SendTime    string `json:"send_time"`    // SendTime is the time the message was sent, typically in DDHHMM format.
}

// PrioritySender holds the priority indicator and sender's address from a SITA message.
type PrioritySender struct {
	Priority string `json:"priority"` // Priority is the message priority code (e.g., "QQ", "FF").
	Sender   string `json:"sender"`   // Sender is the SITA address of the message originator.
}

// TimeReceiver holds the timestamp and receiver's address from a SITA message.
type TimeReceiver struct {
	Time     string `json:"time"`     // Time associated with the receiver line, often similar to SendTime or a processing time.
	Receiver string `json:"receiver"` // Receiver is the SITA address of the message recipient.
}

// Validate checks the SITAHeader, currently focusing on the SendTime format.
// Returns an error if validation fails, otherwise nil.
func (h *SITAHeader) Validate() error {
	log := utils.GetLogger() // Consider if logger is needed here or if errors should just be returned.
	// Validate SendTime format (e.g., DDHHMM)
	if len(h.SendTime) != 6 {
		errMsg := "invalid send_time format, expected DDHHMM"
		log.Errorf("SITAHeader.Validate: %s. Received: %s", errMsg, h.SendTime) // Log includes received value
		return fmt.Errorf("SITAHeader.SendTime: %s", errMsg)                     // Returned error is generic
	}
	return nil
}

// Validate checks the SITA message, currently by validating its embedded SITAHeader.
// Returns an error if the header validation fails, otherwise nil.
func (s *SITA) Validate() error {
	if err := s.Header.Validate(); err != nil {
		return err
	}
	// Add more validation for other SITA fields as needed
	return nil
}

/*
QU TSNZPCA

.HAKUOHU 151234

DISPATCH RELEASE:

HU7670/15MAY ETD1440 B2113/B733

DEP:TSN/ALTN:NIL

ROUTE ALTN:WUH KWL

DEST:HAK/ALTN:NNG SYX

FLT RULE:IFR

TRIP FUEL:9187KGS/20254LBS

TTL FUEL:13600KGS/29983LBS

CREW:YANG XIAOHUI/YANG ZHENGYIN

CREW NUMBER:2/4

SI:CFP AND CAUTION:MXSH 8/FIR

TEL:0898-65756523

FAX:0898-65751587

DSP SIGN:WUKEYONG

PIC SINGN:

(FPL-CHH7670-IS

-B733/M-SDHIRW/S

-ZBTJ1440

-M074S0980 CG A326 VYK A461 LKO R343 LBN/M074S0950 J427 BHY

 W70 NYB

-ZJHK0323 ZGNN ZJSY

-EET/ZHWH0038 ZGZU0144 ZJSA0307

 REG/B2113 SEL/DGEH

 RMK/ACAS EQPT)

NNNN


*/
