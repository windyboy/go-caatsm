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
	// StartIndicator     string      `json:"startIndicator"`               // 电报开始标识: The start of the message indicator (e.g., 'ZCZC').
	Uuid               string      `json:"uuid"`
	MessageID          string      `json:"messageId"`                    // 信息ID: The message ID (e.g., 'TMQ1324').
	DateTime           string      `json:"dateTime"`                     // 日期时间: The date and time of the message (e.g., '150631').
	PriorityIndicator  string      `json:"priorityIndicator"`            // 优先级标识: The priority level of the message (e.g., 'FF').
	PrimaryAddress     string      `json:"primaryAddress"`               // 主要地址: The primary recipient address (e.g., 'ZBTJZPZX').
	SecondaryAddresses []string    `json:"secondaryAddresses,omitempty"` // 次要地址: Additional recipient addresses (e.g., ['150630', 'ZBACZQZX']).
	Originator         string      `json:"originator,omitempty"`         // 发件人: The sender of the message.
	OriginatorDateTime string      `json:"originatorDateTime,omitempty"` // 发件日期时间: The date and time when the originator sent the message.
	Category           string      `json:"category,omitempty"`           // 类别: The category of the message.
	BodyAndFooter      string      `json:"bodyAndFooter,omitempty"`      // 正文和页脚: The body and footer of the message (e.g., 'CALLSIGN/ABC123\nFPL/AB1234-AB\n...').
	BodyData           interface{} `json:"bodyData,omitempty"`           // 正文数据: Parsed body data.
	ReceivedAt         time.Time   `json:"receivedAt"`                   // 接收时间: The time when the message was received.
	ParsedAt           time.Time   `json:"parsedAt,omitempty"`           // 解析时间: The time when the message was parsed.
	DispatchedAt       time.Time   `json:"dispatchedAt,omitempty"`       // 分发时间: The time when the message was dispatched.
	NeedDispatch       bool        `json:"needDispatch"`                 // 需要分发: Indicates if the message needs to be dispatched.
}

// NewParsedMessage initializes a ParsedMessage with default values
func NewParsedMessage() *ParsedMessage {
	return &ParsedMessage{
		SecondaryAddresses: []string{},
	}
}
