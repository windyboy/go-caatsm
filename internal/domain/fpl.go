package domain

import "fmt"

/*
飞行计划报文（FPL）通常包括以下内容：

    编组 3：电报类别、编号和参考数据
    编组 7：航空器识别标志和 SSR 模式及编码
    编组 8：飞行规则和类型
    编组 9：航机和设备
    编组 10：巡航速度和飞行高度
    编组 13：起飞机场和时间
    编组 15：航路
    编组 16：目的地机场和估计总耗时
    编组 18：其他信息（如需要）
    编组 19：补充信息
*/
/*
Group 3: Telegram category, number, and reference data
Group 7: Aircraft identification and SSR mode and code
Group 8: Flight rules and type
Group 9: Number of aircraft, type of aircraft, and wake turbulence category
Group 10: Equipment and capabilities
Group 13: Departure airport and time
Group 15: Route
Group 16: Destination airport and estimated total elapsed time, alternate destination airport
Group 18: Other information (if needed)
Group 19: Supplementary information (if needed)

*/

/*
MessageType (电报类别): Indicates the type of telegram.
FlightNumber (航班号): Represents the flight number.
ReferenceData (参考数据): Optional field for reference data.
AircraftID (航空器识别标志): Unique identifier of the aircraft.
SSRModeAndCode (SSR 模式及编码): SSR (Secondary Surveillance Radar) mode and code.
FlightRulesAndType (飞行规则和类型): Flight rules and type (e.g., IFR).
AircraftAndEquipment (航机和设备): Aircraft type and equipment.
CruisingSpeedAndLevel (巡航速度和飞行高度): Cruising speed and flight level.
DepartureAirport (起飞机场): ICAO code of the departure airport.
DepartureTime (起飞时间): Departure time in UTC.
Route (航路): Planned route.
DestinationAndTotalTime (目的地机场和估计总耗时): ICAO code of the destination airport and estimated total flight time.
AlternateAirport (目的地备降机场): Optional field for the alternate destination airport.
OtherInfo (其他信息): Optional field for additional relevant information.
SupplementaryInfo (补充信息): Optional field for supplementary information.

*/

/*
(FPL-CCA1532-IS
-A332/H
-SDE3FGHIJ4J5M1RWY/LB101
-ZSSS2035
-K0859S1040 PIAKS G330 PIMOL A539 BTO W82 DOGAR
-ZBAA0153 ZBYN
-PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN RMK/TCAS EQUIPPED)

*/

// FPL 电报体中的飞行计划报文结构
type FPL struct {
	Category                string `json:"category"`                     // 电报类别
	FlightNumber            string `json:"flight_number"`                // 航班号
	ReferenceData           string `json:"reference_data,omitempty"`     // 参考数据（可选）
	AircraftID              string `json:"aircraft_id"`                  // 航空器识别标志
	SSRModeAndCode          string `json:"ssr_mode_and_code"`            // SSR 模式及编码
	FlightRulesAndType      string `json:"flight_rules_and_type"`        // 飞行规则和类型
	AircraftAndEquipment    string `json:"aircraft_and_equipment"`       // 航机和设备
	CruisingSpeedAndLevel   string `json:"cruising_speed_and_level"`     // 巡航速度和飞行高度
	DepartureAirport        string `json:"departure_airport"`            // 起飞机场
	DepartureTime           string `json:"departure_time"`               // 起飞时间
	Route                   string `json:"route"`                        // 航路
	DestinationAndTotalTime string `json:"destination_and_total_time"`   // 目的地机场和估计总耗时
	AlternateAirport        string `json:"alternate_airport,omitempty"`  // 目的地备降机场（可选）
	OtherInfo               string `json:"other_info,omitempty"`         // 其他信息（可选）
	SupplementaryInfo       string `json:"supplementary_info,omitempty"` // 补充信息（可选）
}

// Validate validates the FPL struct fields
func (f *FPL) Validate() error {
	if f.Category == "" {
		return fmt.Errorf("telegram category is required")
	}
	if f.FlightNumber == "" {
		return fmt.Errorf("flight number is required")
	}
	if f.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if f.SSRModeAndCode == "" {
		return fmt.Errorf("ssr mode and code is required")
	}
	if f.FlightRulesAndType == "" {
		return fmt.Errorf("flight rules and type is required")
	}
	if f.AircraftAndEquipment == "" {
		return fmt.Errorf("aircraft and equipment is required")
	}
	if f.CruisingSpeedAndLevel == "" {
		return fmt.Errorf("cruising speed and level is required")
	}
	if f.DepartureAirport == "" {
		return fmt.Errorf("departure airport is required")
	}
	if f.DepartureTime == "" {
		return fmt.Errorf("departure time is required")
	}
	if f.Route == "" {
		return fmt.Errorf("route is required")
	}
	if f.DestinationAndTotalTime == "" {
		return fmt.Errorf("destination and total time is required")
	}
	return nil
}
