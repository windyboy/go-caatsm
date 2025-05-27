package domain // Already has package comment

import "fmt"

/*
航班计划修改报文（CHG）一般包括以下内容：

    编组 3：电报类别、编号和参考数据
    编组 7：航空器识别标志和 SSR 模式及编码
    编组 13：起飞机场和时间
    编组 16：目的地机场和估计总耗时，目的地备降机场
    编组 18：其他信息（如需要）
    编组 22：修改部分
*/

/*
Explanation of Each Field
MessageType (电报类别):

Field: MessageType
Description: Indicates the type of telegram (e.g., "CHG").
AircraftID (航空器识别标志):

Field: AircraftID
Description: Unique identifier of the aircraft.
SSRModeAndCode (SSR 模式及编码):

Field: SSRModeAndCode
Description: SSR (Secondary Surveillance Radar) mode and code.
DepartureAirport (起飞机场):

Field: DepartureAirport
Description: ICAO code of the departure airport.
DepartureTime (起飞时间):

Field: DepartureTime
Description: Departure time in UTC.
ArrivalAirport (到达机场):

Field: ArrivalAirport
Description: ICAO code of the arrival airport.
ArrivalTime (到达时间):

Field: ArrivalTime
Description: Estimated arrival time in UTC.
OtherInfo (其他信息):

Field: OtherInfo
Description: Optional field for any additional relevant information.
ChangePart (修改部分):

Field: ChangePart
Description: Indicates the part of the flight plan that is being changed.
*/

/*
(CHG-CCA5678-IS
-B6513
-A1234
-ZBTJ1200
-ZGGG1335
-NEW ROUTE VIA PIAKS G330 PIMOL
-Change reason or additional information)

*/

// CHG represents a Change message (航班计划修改报文).
// It is used to notify changes to a previously submitted flight plan.
type CHG struct {
	Category             string `json:"category"`                         // Category is the message type, typically "CHG". (电报类别)
	AircraftID           string `json:"aircraft_id"`                      // AircraftID is the unique identifier of the aircraft. (航空器识别标志)
	SSRModeAndCode       string `json:"ssr_mode_and_code"`                // SSRModeAndCode is the SSR mode and code. (SSR 模式及编码)
	DepartureAirport     string `json:"departure_airport"`                // DepartureAirport is the ICAO code of the departure airport. (起飞机场)
	DepartureTime        string `json:"departure_time"`                   // DepartureTime is the original or new departure time (typically HHMMSS or similar format). (起飞时间)
	ArrivalAirport       string `json:"arrival_airport"`                  // ArrivalAirport is the ICAO code of the arrival airport. (到达机场)
	ArrivalTime          string `json:"arrival_time"`                     // ArrivalTime is the original or new estimated arrival time (typically HHMMSS or similar format). (到达时间)
	EstimatedElapsedTime string `json:"estimated_elapsed_time"`           // EstimatedElapsedTime is the estimated total flight time. (估计总耗时)
	AlternateAirport     string `json:"alternate_airport,omitempty"`      // AlternateAirport is the alternate destination airport (optional). (目的地备降机场)
	OtherInfo            string `json:"other_info,omitempty"`             // OtherInfo provides a field for any additional relevant information (optional). (其他信息)
	ChangePart           string `json:"change_part"`                      // ChangePart describes the specific part of the flight plan that is being changed. (修改部分)
}

// Validate checks if the mandatory fields of the CHG message are present.
// Returns an error if any mandatory field is empty, otherwise nil.
func (c *CHG) Validate() error {
	if c.Category == "" {
		return fmt.Errorf("CHG.Category: category is required")
	}
	if c.AircraftID == "" {
		return fmt.Errorf("CHG.AircraftID: aircraft id is required")
	}
	if c.SSRModeAndCode == "" {
		return fmt.Errorf("CHG.SSRModeAndCode: ssr mode and code is required")
	}
	if c.DepartureAirport == "" {
		return fmt.Errorf("CHG.DepartureAirport: departure airport is required")
	}
	if c.DepartureTime == "" {
		return fmt.Errorf("CHG.DepartureTime: departure time is required")
	}
	if c.ArrivalAirport == "" {
		return fmt.Errorf("CHG.ArrivalAirport: arrival airport is required")
	}
	if c.ArrivalTime == "" {
		return fmt.Errorf("CHG.ArrivalTime: arrival time is required")
	}
	if c.EstimatedElapsedTime == "" {
		return fmt.Errorf("CHG.EstimatedElapsedTime: estimated elapsed time is required")
	}
	if c.ChangePart == "" {
		return fmt.Errorf("CHG.ChangePart: change part is required")
	}
	return nil
}
