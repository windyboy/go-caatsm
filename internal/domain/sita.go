package domain

import (
	"caatsm/pkg/utils"
	"fmt"
	"time"
)

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

// SITA defines the structure of a SITA telegram
type SITA struct {
	Header            SITAHeader     `json:"header"`          // Header information of the telegram
	PriorityAndSender PrioritySender `json:"priority_sender"` // Priority and sender information
	TimeAndReceiver   TimeReceiver   `json:"time_receiver"`   // Time and receiver information
	Text              string         `json:"text"`            // Content of the telegram
	ReceivedTime      time.Time      `json:"received_time"`   // Time the telegram was received
	Category          string         `json:"category"`        // Category of the telegram
	BodyData          interface{}    `json:"body_data"`       // Additional body data
}

// SITAHeader defines the header of a SITA telegram
type SITAHeader struct {
	StartSignal string `json:"start_signal"` // Start signal indicating the beginning of the telegram
	SendID      string `json:"send_id"`      // Sending ID uniquely identifying the telegram
	SendTime    string `json:"send_time"`    // Sending time in the format DDHHMM
}

// PrioritySender defines priority and sender address
type PrioritySender struct {
	Priority string `json:"priority"` // Priority level
	Sender   string `json:"sender"`   // Sending address
}

// TimeReceiver defines time and receiver address
type TimeReceiver struct {
	Time     string `json:"time"`     // Time of the telegram
	Receiver string `json:"receiver"` // Receiving address
}

func (h *SITAHeader) Validate() error {
	log := utils.GetLogger()
	// Validate SendTime format (e.g., DDHHMM)
	if len(h.SendTime) != 6 {
		err := "invalid send_time format"

		log.Errorf("error validating send_time: %v", err)
		return fmt.Errorf(err)
	}
	return nil
}

func (s *SITA) Validate() error {
	if err := s.Header.Validate(); err != nil {
		return err
	}
	// Add more validation as needed
	return nil
}
