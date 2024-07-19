package domain

import "fmt"

/*
ARR 报文的规范和组成如下：

    编组 3：电报类别、编号和参考数据
    编组 7：航空器识别标志和 SSR 模式及编码
    编组 13：起飞机场和时间
    编组 16：目的地机场和估计总耗时，目的地备降机场
    编组 18：其他信息（如需要）
*/

/*
TelegramCategory: Added for the telegram category (from Group 3).
FlightNumber: Added for the flight number (from Group 3).
ReferenceData: Added for the reference data (from Group 3).
AircraftID: Kept as it was for the aircraft identification (from Group 7).
SSRModeAndCode: Kept as it was for the SSR mode and code (from Group 7).
DepartureAirport: Kept as it was for the departure airport (from Group 13).
DepartureTime: Added for the departure time (from Group 13).
ArrivalAirport: Kept as it was for the arrival airport (from Group 16).
EstimatedElapsedTime: Added for the estimated total elapsed time (from Group 16).
AlternateAirport: Added for the alternate destination airport (from Group 16).
OtherInfo: Kept as it was for any other information (from Group 18).
*/

// ARR 电报体中的到达报文结构
type ARR struct {
	Category             string `json:"category"`               // 电报类别
	AircraftID           string `json:"aircraft_id"`            // 航空器识别标志
	SSRModeAndCode       string `json:"ssr_mode_and_code"`      // SSR 模式及编码（可选）
	DepartureAirport     string `json:"departure_airport"`      // 起飞机场
	DepartureTime        string `json:"departure_time"`         // 起飞时间
	ArrivalAirport       string `json:"arrival_airport"`        // 到达机场
	ArrivalTime          string `json:"arrival_time"`           // 到达时间
	EstimatedElapsedTime string `json:"estimated_elapsed_time"` // 估计总耗时（可选）
	AlternateAirport     string `json:"alternate_airport"`      // 目的地备降机场（可选）
	OtherInfo            string `json:"other_info"`             // 其他信息（可选）
}

// Validate validates the ARR struct fields
func (a *ARR) Validate() error {
	if a.Category == "" {
		return fmt.Errorf("category is required")
	}
	if a.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
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
