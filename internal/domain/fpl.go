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
Example message:
(FPL-JAE7433-IS
-B744/H-SXIRPZJWY/S
-ZBTJ1755
-K0926S0920 CG A326 VYK W80 HUR B339 GM A575 MANSA/K0919S0980 A575
 INTIK/K0917S0960 A575 UDA DCT BULAG A200 HATGA/K0900S1060 A308
 LARNA DCT RATKO A307 KUMOD R497 TODES B228 ZJ R22 KTL R30 SPB
 B141 RANVA/N0485F360 UP863 DEREX UP739 KOLJA UN746 GORPI UZ80
 TILAV UL87 TADUV T173 GED GED2W
-EDDF0948 EDDK
-EET/ZMUB0100 UNKL0236 UNWW0332 UNNT0332 USRR0447 USHH0507
 USSS0535 UUYY0602 ULKK0634 ULWW0653 ULLL0720 EETT0748 EVRR0815
 ESAA0821 EPWW0848 EDUU0900
 REG/B2422 SEL/JLAD OPR/JADE CARGO DAT/S RVR/200
 NAV/RNAV1 RNAV5 RNP4
 RMK/AGCS EQUIPPED
 ACARS EQUIPPED/TCAS EQUIPPED/FOREIGN PILOT
 E/1148 P/TBN R/UV S/M J/LF D/1 15 C YELLOW
 A/WHITE GREEN)
*/

// FPL represents the structure of a Flight Plan message in the FPL telegram body
type FPL struct {
	Category                string `json:"category"`                      // 电报类别: The category of the telegram (e.g., 'FPL' for Flight Plan).
	FlightNumber            string `json:"flight_number"`                 // 航班号: The flight number (e.g., 'JAE7433').
	ReferenceData           string `json:"reference_data,omitempty"`      // 参考数据（可选）: Reference data, if applicable.
	AircraftID              string `json:"aircraft_id"`                   // 航空器识别标志: The aircraft identification (e.g., 'B744/H').
	SSRModeAndCode          string `json:"ssr_mode_and_code"`             // SSR 模式及编码: The SSR mode and code (e.g., 'SXIRPZJWY/S').
	FlightRulesAndType      string `json:"flight_rules_and_type"`         // 飞行规则和类型: Flight rules and type (e.g., 'IS').
	CruisingSpeedAndLevel   string `json:"cruising_speed_and_level"`      // 巡航速度和飞行高度: Cruising speed and flight level (e.g., 'K0926S0920').
	DepartureAirport        string `json:"departure_airport"`             // 起飞机场: Departure airport code (e.g., 'ZBTJ').
	DepartureTime           string `json:"departure_time"`                // 起飞时间: Departure time (e.g., '1755').
	Route                   string `json:"route"`                         // 航路: The flight route (e.g., 'CG A326 VYK W80 HUR ... GED2W').
	DestinationAndTotalTime string `json:"destination_and_total_time"`    // 目的地机场和估计总耗时: Destination airport and estimated total time (e.g., 'EDDF0948').
	AlternateAirport        string `json:"alternate_airport,omitempty"`   // 目的地备降机场（可选）: Alternate airport (e.g., 'EDDK').
	OtherInfo               string `json:"other_info,omitempty"`          // 其他信息（可选）: Other information.
	SupplementaryInfo       string `json:"supplementary_info,omitempty"`  // 补充信息（可选）: Supplementary information.
	EstimatedArrivalTime    string `json:"estimated_arrival_time"`        // 预计到达时间: Estimated time of arrival (e.g., '0948').
	PBN                     string `json:"pbn"`                           // 性能导航: Performance-based navigation equipment (e.g., 'A1B2B3B4B5D1L1').
	NavigationEquipment     string `json:"navigation_equipment"`          // 导航设备: Navigation equipment (e.g., 'NAV/ABAS').
	EstimatedElapsedTime    string `json:"estimated_elapsed_time"`        // 估计飞行时间: Estimated elapsed time (e.g., 'EET/ZMUB0100').
	SELCALCode              string `json:"selcal_code"`                   // SELCAL代码: SELCAL code (e.g., 'JLAD').
	PerformanceCategory     string `json:"performance_category"`          // 性能类别: Aircraft performance category (e.g., 'C').
	RerouteInformation      string `json:"reroute_information,omitempty"` // 重航信息（可选）: Reroute information (e.g., 'RIF/FRT N640 ZBYN').
	Remarks                 string `json:"remarks,omitempty"`             // 备注（可选）: Remarks (e.g., 'RMK/TCAS EQUIPPED').
}

// Validate validates the FPL struct fields
func (f *FPL) Validate() error {
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
