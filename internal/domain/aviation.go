package domain

import "time"

// An aviation message typically contains various fields that are crucial for air traffic management and communication.
//These fields include identifiers, date and time, priority indicators, addresses,
//and additional information such as call signs, flight plans, route details, altitude, speed, position,
//emergency indicators, report types, supplementary information, filing times, originator indicators, service information,
//and navigation aid details. The ParsedMessage struct is designed to encapsulate all these details in a structured format.

/*
Example message 1:
ZCZC TMQ1324 150631
FF ZBTJZPZX
150630 ZBACZQZX
CALLSIGN/ABC123
FPL/AB1234-AB
ROUTE/NOR1.DCT
ALTITUDE/35000FT
SPEED/450KT
POSITION/N55W011
EMG/N
RPT/POS
SUP/Additional info
FLIGHTPLANID/FP1234
FILTIME/150600
ORIIND/AB12
SERINFO/Service info
NAVINFO/NAV details

Field values:
StartIndicator: "ZCZC".
MessageID: "TMQ1324".
DateTime: "150631".
PriorityIndicator: "FF".
PrimaryAddress: "ZBTJZPZX".
SecondaryAddresses: ["150630", "ZBACZQZX"].
Originator: "".
OriginatorDateTime: "".
Category: "".
BodyAndFooter: "CALLSIGN/ABC123\nFPL/AB1234-AB\nROUTE/NOR1.DCT\nALTITUDE/35000FT\nSPEED/450KT\nPOSITION/N55W011\nEMG/N\nRPT/POS\nSUP/Additional info\nFLIGHTPLANID/FP1234\nFILTIME/150600\nORIIND/AB12\nSERINFO/Service info\nNAVINFO/NAV details".
BodyData: nil.
ReceivedAt: time.Time{}.
ParsedAt: time.Time{}.
DispatchedAt: time.Time{}.
NeedDispatch: false.
*/

/*
Example message 2:
ZCZC XMP4567 120915
DD KLAXZPZX
120914 KSFOZQZX
CALLSIGN/DEF456
FPL/CD5678-DC
ROUTE/NOR2.DCT
ALTITUDE/36000FT
SPEED/500KT
POSITION/N54W012
EMG/Y
RPT/WX
SUP/Weather related info
FLIGHTPLANID/FP5678
FILTIME/120900
ORIIND/CD34
SERINFO/Service related info
NAVINFO/Navigation details

Field values:
StartIndicator: "ZCZC".
MessageID: "XMP4567".
DateTime: "120915".
PriorityIndicator: "DD".
PrimaryAddress: "KLAXZPZX".
SecondaryAddresses: ["120914", "KSFOZQZX"].
Originator: "".
OriginatorDateTime: "".
Category: "".
BodyAndFooter: "CALLSIGN/DEF456\nFPL/CD5678-DC\nROUTE/NOR2.DCT\nALTITUDE/36000FT\nSPEED/500KT\nPOSITION/N54W012\nEMG/Y\nRPT/WX\nSUP/Weather related info\nFLIGHTPLANID/FP5678\nFILTIME/120900\nORIIND/CD34\nSERINFO/Service related info\nNAVINFO/Navigation details".
BodyData: nil.
ReceivedAt: time.Time{}.
ParsedAt: time.Time{}.
DispatchedAt: time.Time{}.
NeedDispatch: false.
*/

// ParsedMessage holds the parsed data from an aviation message
type ParsedMessage struct {
	StartIndicator     string      `json:"startIndicator"`               // "ZCZC".
	MessageID          string      `json:"messageId"`                    // "TMQ1324".
	DateTime           string      `json:"dateTime"`                     // "150631".
	PriorityIndicator  string      `json:"priorityIndicator"`            // "FF".
	PrimaryAddress     string      `json:"primaryAddress"`               // "ZBTJZPZX".
	SecondaryAddresses []string    `json:"secondaryAddresses,omitempty"` // ["150630", "ZBACZQZX"].
	Originator         string      `json:"originator,omitempty"`         // "".
	OriginatorDateTime string      `json:"originatorDateTime,omitempty"` // "".
	Category           string      `json:"category,omitempty"`           // "".
	BodyAndFooter      string      `json:"bodyAndFooter,omitempty"`      // "CALLSIGN/ABC123\nFPL/AB1234-AB\nROUTE/NOR1.DCT\nALTITUDE/35000FT\nSPEED/450KT\nPOSITION/N55W011\nEMG/N\nRPT/POS\nSUP/Additional info\nFLIGHTPLANID/FP1234\nFILTIME/150600\nORIIND/AB12\nSERINFO/Service info\nNAVINFO/NAV details".
	BodyData           interface{} `json:"bodyData,omitempty"`           // nil.
	ReceivedAt         time.Time   `json:"receivedAt"`                   // time.Time{}.
	ParsedAt           time.Time   `json:"parsedAt,omitempty"`           // time.Time{}.
	DispatchedAt       time.Time   `json:"dispatchedAt,omitempty"`       // time.Time{}.
	NeedDispatch       bool        `json:"needDispatch"`                 // false.
}

// NewParsedMessage initializes a ParsedMessage with default values
func NewParsedMessage() *ParsedMessage {
	return &ParsedMessage{
		SecondaryAddresses: []string{},
	}
}
