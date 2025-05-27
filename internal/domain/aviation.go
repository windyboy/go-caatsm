// Package domain contains the core data structures (models) used throughout the caatsm application.
// These structures represent various aviation message types, scheduling information,
// and general message parsing results.
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

// ParsedMessage holds the structured data extracted from a raw aviation message.
// It includes header information, the raw content, and potentially structured body data
// if parsing was successful.
type ParsedMessage struct {
	StartIndicator     string      `json:"startIndicator"`               // StartIndicator is the beginning marker of the message (e.g., "ZCZC"). (电报开始标识)
	Uuid               string      `json:"uuid"`                         // Uuid is a unique identifier assigned to the message upon processing.
	MessageID          string      `json:"messageId"`                    // MessageID is the identifier extracted from the message itself (e.g., "TMQ1324"). (信息ID)
	DateTime           string      `json:"dateTime"`                     // DateTime is the date and time group from the message header (e.g., "150631"). (日期时间)
	PriorityIndicator  string      `json:"priorityIndicator"`            // PriorityIndicator signifies the message's urgency (e.g., "FF", "GG"). (优先级标识)
	PrimaryAddress     string      `json:"primaryAddress"`               // PrimaryAddress is the main recipient address. (主要地址)
	SecondaryAddresses string      `json:"secondaryAddresses,omitempty"` // SecondaryAddresses are additional recipient addresses, stored as a single string. (次要地址)
	Originator         string      `json:"originator,omitempty"`         // Originator is the sender's address or identifier. (发件人)
	OriginatorDateTime string      `json:"originatorDateTime,omitempty"` // OriginatorDateTime is the timestamp from the originator line. (发件日期时间)
	Category           string      `json:"category,omitempty"`           // Category is the identified type of the aviation message (e.g., "ARR", "FPL"). (类别)

	// Body is the extracted body portion of the message, used by the parser to create specific message types (e.g., FPL, ARR).
	// This field is primarily for internal parser use and not typically marshalled to JSON directly,
	// as its structured representation is in BodyData.
	Body string

	// Content represents the complete raw content of the received message. (正文)
	Content string `json:"content,omitempty"`

	BodyData     interface{} `json:"bodyData,omitempty"`     // BodyData holds the parsed, structured data specific to the message Category (e.g., an ARR struct, FPL struct). (正文数据)
	ReceivedAt   time.Time   `json:"receivedAt"`             // ReceivedAt is the timestamp when the message was received by the system. (接收时间)
	ParsedAt     time.Time   `json:"parsedAt,omitempty"`     // ParsedAt is the timestamp when the message was successfully parsed. (解析时间)
	DispatchedAt time.Time   `json:"dispatchedAt,omitempty"` // DispatchedAt is the timestamp when the message was dispatched (e.g., to another system). (分发时间)
	NeedDispatch bool        `json:"needDispatch"`           // NeedDispatch indicates if the message should be further dispatched. (需要分发)
	Parsed       bool        `json:"parsed"`                 // Parsed indicates whether the BodyData has been successfully populated. (解析)
	Comments     string      `json:"comments,omitempty"`     // Comments holds any remarks or error messages generated during parsing or processing. (备注)
}

// NewParsedMessage initializes a new ParsedMessage with default values,
// particularly setting Parsed to false.
func NewParsedMessage() *ParsedMessage {
	return &ParsedMessage{
		Parsed: false,
	}
}

// ToString provides a brief string representation of the ParsedMessage,
// typically used for quick logging or identification.
func (message *ParsedMessage) ToString() string {
	return message.MessageID + " " + message.Category + " " + message.Originator
}
