package domain

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

// ALN 电报体中的预警报文结构
type ALN struct {
	Category           string `json:"category"`              // 电报类别
	AircraftID         string `json:"aircraft_id"`           // 航空器识别标志
	SSRModeAndCode     string `json:"ssr_mode_and_code"`     // SSR 模式及编码
	FlightRulesAndType string `json:"flight_rules_and_type"` // 飞行规则和类型
	DepartureAirport   string `json:"departure_airport"`     // 起飞机场
	DepartureTime      string `json:"departure_time"`        // 起飞时间
	ArrivalAirport     string `json:"arrival_airport"`       // 到达机场
	ArrivalTime        string `json:"arrival_time"`          // 到达时间
	OtherInfo          string `json:"other_info,omitempty"`  // 其他信息 (optional)
}

// Validate validates the ALN struct fields
func (a *ALN) Validate() error {
	if a.Category == "" {
		return fmt.Errorf("telegram category is required")
	}
	if a.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if a.SSRModeAndCode == "" {
		return fmt.Errorf("ssr mode and code is required")
	}
	if a.FlightRulesAndType == "" {
		return fmt.Errorf("flight rules and type is required")
	}
	if a.DepartureAirport == "" {
		return fmt.Errorf("departure airport is required")
	}
	if a.DepartureTime == "" {
		return fmt.Errorf("departure time is required")
	}
	if a.ArrivalAirport == "" {
		return fmt.Errorf("arrival airport is required")
	}
	if a.ArrivalTime == "" {
		return fmt.Errorf("arrival time is required")
	}
	return nil
}
