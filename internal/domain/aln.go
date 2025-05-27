package domain // Already has package comment from aviation.go

import "fmt"

/*
预警报文（ALN）通常包括以下内容：

    编组 3：电报类别、编号和参考数据
    编组 7：航空器识别标志和 SSR 模式及编码
    编组 8：飞行规则和类型
    编组 13：起飞机场和时间
    编组 16：目的地机场和估计总耗时，目的地备降机场
    编组 18：其他信息（如需要）
*/

/*
MessageType (电报类别): Indicates the type of telegram (e.g., "ALN").
AircraftID (航空器识别标志): Unique identifier of the aircraft.
SSRModeAndCode (SSR 模式及编码): SSR (Secondary Surveillance Radar) mode and code.
FlightRulesAndType (飞行规则和类型): Flight rules and type (e.g., IFR).
DepartureAirport (起飞机场): ICAO code of the departure airport.
DepartureTime (起飞时间): Departure time in UTC.
ArrivalAirport (到达机场): ICAO code of the arrival airport.
ArrivalTime (到达时间): Estimated arrival time in UTC.
OtherInfo (其他信息): Optional field for any additional relevant information.

*/
/*
(ALN-CCA1234-IS
-B6513
-A1234
-IFR
-ZBTJ1200
-ZGGG1335
-ESTIMATED TIME EN ROUTE 01:35
-Additional information)

*/

// ALN represents an Alert message (预警报文).
// It contains details about a flight alert, including aircraft identification,
// flight rules, departure/arrival information, and SSR data.
type ALN struct {
	Category           string `json:"category"`              // Category is the message type, typically "ALN". (电报类别)
	AircraftID         string `json:"aircraft_id"`           // AircraftID is the unique identifier of the aircraft. (航空器识别标志)
	SSRModeAndCode     string `json:"ssr_mode_and_code"`     // SSRModeAndCode is the Secondary Surveillance Radar mode and code. (SSR 模式及编码)
	FlightRulesAndType string `json:"flight_rules_and_type"` // FlightRulesAndType indicates the flight rules (e.g., IFR) and type of flight. (飞行规则和类型)
	DepartureAirport   string `json:"departure_airport"`     // DepartureAirport is the ICAO code of the departure airport. (起飞机场)
	DepartureTime      string `json:"departure_time"`        // DepartureTime is the departure time in UTC (typically HHMMSS or similar format). (起飞时间)
	ArrivalAirport     string `json:"arrival_airport"`       // ArrivalAirport is the ICAO code of the arrival airport. (到达机场)
	ArrivalTime        string `json:"arrival_time"`          // ArrivalTime is the estimated arrival time in UTC (typically HHMMSS or similar format). (到达时间)
	OtherInfo          string `json:"other_info,omitempty"`  // OtherInfo provides a field for any additional relevant information (optional). (其他信息)
}

// Validate checks if the mandatory fields of the ALN message are present.
// Returns an error if any mandatory field is empty, otherwise nil.
func (a *ALN) Validate() error {
	if a.Category == "" {
		return fmt.Errorf("ALN.Category: telegram category is required")
	}
	if a.AircraftID == "" {
		return fmt.Errorf("ALN.AircraftID: aircraft id is required")
	}
	if a.SSRModeAndCode == "" {
		return fmt.Errorf("ALN.SSRModeAndCode: ssr mode and code is required")
	}
	if a.FlightRulesAndType == "" {
		return fmt.Errorf("ALN.FlightRulesAndType: flight rules and type is required")
	}
	if a.DepartureAirport == "" {
		return fmt.Errorf("ALN.DepartureAirport: departure airport is required")
	}
	if a.DepartureTime == "" {
		return fmt.Errorf("ALN.DepartureTime: departure time is required")
	}
	if a.ArrivalAirport == "" {
		return fmt.Errorf("ALN.ArrivalAirport: arrival airport is required")
	}
	if a.ArrivalTime == "" {
		return fmt.Errorf("ALN.ArrivalTime: arrival time is required")
	}
	return nil
}
