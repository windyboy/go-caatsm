package domain // Already has package comment

import "fmt"

/*
航班计划变更报文（CPL）通常包括以下内容：

    编组 3：电报类别、编号和参考数据
    编组 7：航空器识别标志和 SSR 模式及编码
    编组 8：飞行规则和类型
    编组 9：航机和设备
    编组 10：巡航速度和飞行高度
    编组 13：起飞机场和时间
    编组 15：航路
    编组 16：目的地机场和总时间，目的地备降机场
    编组 18：其他信息（如需要）
*/
/*
MessageType (电报类别):

Field: MessageType
Description: Indicates the type of telegram (e.g., "CPL").
AircraftID (航空器识别标志):

Field: AircraftID
Description: Unique identifier of the aircraft.
SSRModeAndCode (SSR 模式及编码):

Field: SSRModeAndCode
Description: SSR (Secondary Surveillance Radar) mode and code.
FlightRulesAndType (飞行规则和类型):

Field: FlightRulesAndType
Description: Flight rules and type (e.g., IFR).
AircraftAndEquipment (航机和设备):

Field: AircraftAndEquipment
Description: Aircraft type and equipment.
CruisingSpeedAndLevel (巡航速度和飞行高度):

Field: CruisingSpeedAndLevel
Description: Cruising speed and flight level.
DepartureAirport (起飞机场):

Field: DepartureAirport
Description: ICAO code of the departure airport.
DepartureTime (起飞时间):

Field: DepartureTime
Description: Departure time in UTC.
Route (航路):

Field: Route
Description: Planned route.
DestinationAndTotalTime (目的地机场和总时间):

Field: DestinationAndTotalTime
Description: ICAO code of the destination airport and total elapsed time.
OtherInfo (其他信息):

Field: OtherInfo
Description: Optional field for any additional relevant information.
*/

/*
(CPL-CCA7890-IS
-B6513
-A1234
-IFR
-A332/H
-K0859S1040
-ZBTJ1200
-PIAKS G330 PIMOL A539 BTO W82 DOGAR
-ZGGG0135
-ZBAA
-Additional information)

*/

// CPL represents a Current Flight Plan message (当前飞行计划报文).
// This message type details the current and active flight plan for an aircraft.
type CPL struct {
	Category                string `json:"category"`                    // Category is the message type, typically "CPL". (电报类别)
	AircraftID              string `json:"aircraft_id"`                 // AircraftID is the unique identifier of the aircraft. (航空器识别标志)
	SSRModeAndCode          string `json:"ssr_mode_and_code"`           // SSRModeAndCode is the SSR mode and code. (SSR 模式及编码)
	FlightRulesAndType      string `json:"flight_rules_and_type"`       // FlightRulesAndType indicates the flight rules (e.g., IFR) and type of flight. (飞行规则和类型)
	AircraftAndEquipment    string `json:"aircraft_and_equipment"`      // AircraftAndEquipment specifies the aircraft type and its equipment. (航机和设备)
	CruisingSpeedAndLevel   string `json:"cruising_speed_and_level"`    // CruisingSpeedAndLevel indicates the planned cruising speed and flight level. (巡航速度和飞行高度)
	DepartureAirport        string `json:"departure_airport"`           // DepartureAirport is the ICAO code of the departure airport. (起飞机场)
	DepartureTime           string `json:"departure_time"`              // DepartureTime is the departure time (typically HHMMSS or similar format). (起飞时间)
	Route                   string `json:"route"`                       // Route describes the planned flight path. (航路)
	DestinationAndTotalTime string `json:"destination_and_total_time"`  // DestinationAndTotalTime includes the ICAO code of the destination airport and the total estimated elapsed time. (目的地机场和总时间)
	AlternateAirport        string `json:"alternate_airport,omitempty"` // AlternateAirport is the ICAO code of the alternate destination airport (optional). (目的地备降机场)
	OtherInfo               string `json:"other_info,omitempty"`        // OtherInfo provides a field for any additional relevant information (optional). (其他信息)
}

// Validate checks if the mandatory fields of the CPL message are present.
// Returns an error if any mandatory field is empty, otherwise nil.
func (c *CPL) Validate() error {
	if c.Category == "" {
		return fmt.Errorf("CPL.Category: category is required")
	}
	if c.AircraftID == "" {
		return fmt.Errorf("CPL.AircraftID: aircraft id is required")
	}
	if c.SSRModeAndCode == "" {
		return fmt.Errorf("CPL.SSRModeAndCode: ssr mode and code is required")
	}
	if c.FlightRulesAndType == "" {
		return fmt.Errorf("CPL.FlightRulesAndType: flight rules and type is required")
	}
	if c.AircraftAndEquipment == "" {
		return fmt.Errorf("CPL.AircraftAndEquipment: aircraft and equipment is required")
	}
	if c.CruisingSpeedAndLevel == "" {
		return fmt.Errorf("CPL.CruisingSpeedAndLevel: cruising speed and level is required")
	}
	if c.DepartureAirport == "" {
		return fmt.Errorf("CPL.DepartureAirport: departure airport is required")
	}
	if c.DepartureTime == "" {
		return fmt.Errorf("CPL.DepartureTime: departure time is required")
	}
	if c.Route == "" {
		return fmt.Errorf("CPL.Route: route is required")
	}
	if c.DestinationAndTotalTime == "" {
		return fmt.Errorf("CPL.DestinationAndTotalTime: destination and total time is required")
	}
	return nil
}
