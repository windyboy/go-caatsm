package domain // Already has package comment

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

// FPL represents a Filed Flight Plan message (飞行计划报文).
// It contains comprehensive details about a planned flight.
type FPL struct {
	Category                string `json:"category"`                      // Category is the message type, typically "FPL". (电报类别)
	FlightNumber            string `json:"flight_number"`                 // FlightNumber is the flight identifier (e.g., "JAE7433"). (航班号)
	ReferenceData           string `json:"reference_data,omitempty"`      // ReferenceData provides any reference information if applicable (optional). (参考数据)
	AircraftID              string `json:"aircraft_id"`                   // AircraftID is the aircraft type and wake turbulence category (e.g., "B744/H"). (航空器识别标志 - 通常指机型和尾流)
	SSRModeAndCode          string `json:"ssr_mode_and_code"`             // SSRModeAndCode is the SSR mode and code, often including equipment codes (e.g., "SXIRPZJWY/S"). (SSR 模式及编码 - 通常指设备能力)
	FlightRulesAndType      string `json:"flight_rules_and_type"`         // FlightRulesAndType indicates flight rules (e.g., IFR-"I") and type of flight (e.g., Scheduled-"S"). (飞行规则和类型)
	CruisingSpeedAndLevel   string `json:"cruising_speed_and_level"`      // CruisingSpeedAndLevel is the planned cruising speed and flight level (e.g., "K0926S0920"). (巡航速度和飞行高度)
	DepartureAirport        string `json:"departure_airport"`             // DepartureAirport is the ICAO code of the departure airport (e.g., "ZBTJ"). (起飞机场)
	DepartureTime           string `json:"departure_time"`                // DepartureTime is the estimated off-block time or departure time (typically HHMM or HHMMSS format). (起飞时间)
	Route                   string `json:"route"`                         // Route describes the planned flight path. (航路)
	DestinationAndTotalTime string `json:"destination_and_total_time"`    // DestinationAndTotalTime includes the ICAO code of the destination airport and total estimated elapsed time (e.g., "EDDF0948"). (目的地机场和估计总耗时)
	AlternateAirport        string `json:"alternate_airport,omitempty"`   // AlternateAirport is the ICAO code of the alternate destination airport(s) (optional). (目的地备降机场)
	OtherInfo               string `json:"other_info,omitempty"`          // OtherInfo contains various supplementary details, often structured with sub-fields (e.g., PBN, NAV, REG, EET) (optional). (其他信息 - 通常指FIELD 18)
	SupplementaryInfo       string `json:"supplementary_info,omitempty"`  // SupplementaryInfo (FIELD 19) contains coded or plain language data required by authorities (optional). (补充信息)
	EstimatedArrivalTime    string `json:"estimated_arrival_time"`        // EstimatedArrivalTime is the estimated time of arrival, often derived from DestinationAndTotalTime. (预计到达时间)
	PBN                     string `json:"pbn"`                           // PBN indicates Performance-Based Navigation capabilities (parsed from OtherInfo). (性能导航)
	NavigationEquipment     string `json:"navigation_equipment"`          // NavigationEquipment specifies navigation aids (parsed from OtherInfo). (导航设备)
	EstimatedElapsedTime    string `json:"estimated_elapsed_time"`        // EstimatedElapsedTime is the estimated elapsed time to specific points or FIR boundaries (parsed from OtherInfo EET/). (估计飞行时间)
	SELCALCode              string `json:"selcal_code"`                   // SELCALCode is the aircraft's SELCAL code (parsed from OtherInfo SEL/). (SELCAL代码)
	Register                string `json:"register,omitempty"`            // Register is the aircraft registration mark (parsed from OtherInfo REG/) (optional). (注册号)
	PerformanceCategory     string `json:"performance_category"`          // PerformanceCategory indicates aircraft performance category (parsed from OtherInfo PER/). (性能类别)
	RerouteInformation      string `json:"reroute_information,omitempty"` // RerouteInformation details any reroutes (parsed from OtherInfo RIF/) (optional). (重航信息)
	Remarks                 string `json:"remarks,omitempty"`             // Remarks provides additional plain language remarks (parsed from OtherInfo RMK/) (optional). (备注)
}

// Validate checks if the mandatory fields of the FPL message are present.
// Returns an error if any mandatory field is empty, otherwise nil.
func (f *FPL) Validate() error {
	// Assuming Category is a standard field for FPL messages, similar to other types.
	// If Category is not part of the FPL specific data but rather the envelope, this check might be redundant here.
	if f.Category == "" {
		return fmt.Errorf("FPL.Category: category is required")
	}
	if f.FlightNumber == "" {
		return fmt.Errorf("FPL.FlightNumber: flight number is required")
	}
	if f.AircraftID == "" {
		return fmt.Errorf("FPL.AircraftID: aircraft id is required")
	}
	if f.SSRModeAndCode == "" {
		return fmt.Errorf("FPL.SSRModeAndCode: SSR mode and code is required")
	}
	if f.FlightRulesAndType == "" {
		return fmt.Errorf("FPL.FlightRulesAndType: flight rules and type is required")
	}
	if f.CruisingSpeedAndLevel == "" {
		return fmt.Errorf("FPL.CruisingSpeedAndLevel: cruising speed and level is required")
	}
	if f.DepartureAirport == "" {
		return fmt.Errorf("FPL.DepartureAirport: departure airport is required")
	}
	if f.DepartureTime == "" {
		return fmt.Errorf("FPL.DepartureTime: departure time is required")
	}
	if f.Route == "" {
		return fmt.Errorf("FPL.Route: route is required")
	}
	if f.DestinationAndTotalTime == "" {
		return fmt.Errorf("FPL.DestinationAndTotalTime: destination and total time is required")
	}
	// Note: Fields like PBN, NavigationEquipment, EstimatedElapsedTime, SELCALCode, PerformanceCategory
	// are listed as required in validation. These are often part of Field 18 (OtherInfo).
	// This strict validation implies they must be present and parsed from OtherInfo.
	if f.EstimatedArrivalTime == "" {
		return fmt.Errorf("FPL.EstimatedArrivalTime: estimated arrival time is required")
	}
	if f.PBN == "" {
		return fmt.Errorf("FPL.PBN: PBN information is required")
	}
	if f.NavigationEquipment == "" {
		return fmt.Errorf("FPL.NavigationEquipment: navigation equipment is required")
	}
	if f.EstimatedElapsedTime == "" {
		return fmt.Errorf("FPL.EstimatedElapsedTime: estimated elapsed time (EET) is required")
	}
	if f.SELCALCode == "" {
		return fmt.Errorf("FPL.SELCALCode: SELCAL code is required")
	}
	if f.PerformanceCategory == "" {
		return fmt.Errorf("FPL.PerformanceCategory: performance category is required")
	}
	return nil
}
