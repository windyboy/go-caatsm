package domain // Already has package comment from aviation.go

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

// ARR represents an Arrival message (到达报文).
// It signifies that an aircraft has arrived at its destination.
type ARR struct {
	Category             string `json:"category"`                         // Category is the message type, typically "ARR". (电报类别)
	AircraftID           string `json:"aircraft_id"`                      // AircraftID is the unique identifier of the aircraft. (航空器识别标志)
	SSRModeAndCode       string `json:"ssr_mode_and_code,omitempty"`      // SSRModeAndCode is the SSR mode and code (optional). (SSR 模式及编码)
	DepartureAirport     string `json:"departure_airport"`                // DepartureAirport is the ICAO code of the departure airport. (起飞机场)
	DepartureTime        string `json:"departure_time"`                   // DepartureTime is the departure time (optional for ARR, but often included, typically HHMMSS or similar format). (起飞时间)
	ArrivalAirport       string `json:"arrival_airport"`                  // ArrivalAirport is the ICAO code of the arrival airport. (到达机场)
	ArrivalTime          string `json:"arrival_time"`                     // ArrivalTime is the actual or estimated time of arrival (typically HHMMSS or similar format). (到达时间)
	EstimatedElapsedTime string `json:"estimated_elapsed_time,omitempty"` // EstimatedElapsedTime is the estimated total flight time (optional). (估计总耗时)
	AlternateAirport     string `json:"alternate_airport,omitempty"`      // AlternateAirport is the alternate destination airport (optional). (目的地备降机场)
	OtherInfo            string `json:"other_info,omitempty"`             // OtherInfo provides a field for any additional relevant information (optional). (其他信息)
}

// Validate checks if the mandatory fields of the ARR message are present.
// Returns an error if any mandatory field is empty, otherwise nil.
func (a *ARR) Validate() error {
	if a.Category == "" {
		return fmt.Errorf("ARR.Category: category is required")
	}
	if a.AircraftID == "" {
		return fmt.Errorf("ARR.AircraftID: aircraft id is required")
	}
	if a.DepartureAirport == "" {
		return fmt.Errorf("ARR.DepartureAirport: departure airport is required")
	}
	if a.DepartureTime == "" {
		return fmt.Errorf("ARR.DepartureTime: departure time is required")
	}
	if a.ArrivalAirport == "" {
		return fmt.Errorf("ARR.ArrivalAirport: arrival airport is required")
	}
	if a.ArrivalTime == "" {
		return fmt.Errorf("ARR.ArrivalTime: arrival time is required")
	}
	return nil
}
