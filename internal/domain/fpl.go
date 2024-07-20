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
	Category                string `json:"category"`                      // 电报类别: The category of the telegram (e.g., 'FPL' for Flight Plan).
	FlightNumber            string `json:"flight_number"`                 // 航班号: The flight number (e.g., 'CCA1532').
	ReferenceData           string `json:"reference_data,omitempty"`      // 参考数据（可选）: Reference data, if applicable.
	AircraftID              string `json:"aircraft_id"`                   // 航空器识别标志: The aircraft identification (e.g., 'A332/H').
	SSRModeAndCode          string `json:"ssr_mode_and_code"`             // SSR 模式及编码: The SSR mode and code (e.g., 'SDE3FGHIJ4J5M1RWY/LB101').
	FlightRulesAndType      string `json:"flight_rules_and_type"`         // 飞行规则和类型: Flight rules and type (e.g., 'IS').
	CruisingSpeedAndLevel   string `json:"cruising_speed_and_level"`      // 巡航速度和飞行高度: Cruising speed and flight level (e.g., 'K0859S1040').
	DepartureAirport        string `json:"departure_airport"`             // 起飞机场: Departure airport code (e.g., 'ZSSS').
	DepartureTime           string `json:"departure_time"`                // 起飞时间: Departure time (e.g., '2035').
	Route                   string `json:"route"`                         // 航路: The flight route (e.g., 'PIAKS G330 PIMOL A539 BTO W82 DOGAR').
	DestinationAndTotalTime string `json:"destination_and_total_time"`    // 目的地机场和估计总耗时: Destination airport and estimated total time (e.g., 'ZBAA0153').
	AlternateAirport        string `json:"alternate_airport,omitempty"`   // 目的地备降机场（可选）: Alternate airport (e.g., 'ZBYN').
	OtherInfo               string `json:"other_info,omitempty"`          // 其他信息（可选）: Other information.
	SupplementaryInfo       string `json:"supplementary_info,omitempty"`  // 补充信息（可选）: Supplementary information.
	SurveillanceEquipment   string `json:"surveillance_equipment"`        // 监视设备信息: Surveillance equipment information (e.g., 'SDE3FGHIJ4J5M1RWY').
	EstimatedArrivalTime    string `json:"estimated_arrival_time"`        // 预计到达时间: Estimated time of arrival (e.g., '0153').
	PBN                     string `json:"pbn"`                           // 性能导航: Performance-based navigation equipment (e.g., 'A1B2B3B4B5D1L1').
	NavigationEquipment     string `json:"navigation_equipment"`          // 导航设备: Navigation equipment (e.g., 'NAV/ABAS').
	EstimatedElapsedTime    string `json:"estimated_elapsed_time"`        // 估计飞行时间: Estimated elapsed time (e.g., 'EET/ZBPE0112').
	SELCALCode              string `json:"selcal_code"`                   // SELCAL代码: SELCAL code (e.g., 'KMAL').
	PerformanceCategory     string `json:"performance_category"`          // 性能类别: Aircraft performance category (e.g., 'C').
	RerouteInformation      string `json:"reroute_information,omitempty"` // 重航信息（可选）: Reroute information (e.g., 'RIF/FRT N640 ZBYN').
	Remarks                 string `json:"remarks,omitempty"`             // 备注（可选）: Remarks (e.g., 'RMK/TCAS EQUIPPED').
}

// Validate validates the FPL struct fields
func (f *FPL) Validate() error {
	//add validation logic here
	if f.FlightNumber == "" {
		return fmt.Errorf("flight number is required")
	}
	if f.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if f.SSRModeAndCode == "" {
		return fmt.Errorf("SSR mode and code is required")
	}
	if f.FlightRulesAndType == "" {
		return fmt.Errorf("flight rules and type is required")
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
	if f.SurveillanceEquipment == "" {
		return fmt.Errorf("surveillance equipment is required")
	}
	if f.EstimatedArrivalTime == "" {
		return fmt.Errorf("estimated arrival time is required")
	}
	if f.PBN == "" {
		return fmt.Errorf("PBN is required")
	}
	if f.NavigationEquipment == "" {
		return fmt.Errorf("navigation equipment is required")
	}
	if f.EstimatedElapsedTime == "" {
		return fmt.Errorf("estimated elapsed time is required")
	}
	if f.SELCALCode == "" {
		return fmt.Errorf("SELCAL code is required")
	}
	if f.PerformanceCategory == "" {
		return fmt.Errorf("performance category is required")
	}
	return nil
}
