package domain

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

// CHG 电报体中的航班计划修改报文结构
type CHG struct {
	Category             string `json:"category"`               // 电报类别
	AircraftID           string `json:"aircraft_id"`            // 航空器识别标志
	SSRModeAndCode       string `json:"ssr_mode_and_code"`      // SSR 模式及编码
	DepartureAirport     string `json:"departure_airport"`      // 起飞机场
	DepartureTime        string `json:"departure_time"`         // 起飞时间
	ArrivalAirport       string `json:"arrival_airport"`        // 到达机场
	ArrivalTime          string `json:"arrival_time"`           // 到达时间
	EstimatedElapsedTime string `json:"estimated_elapsed_time"` // 估计总耗时
	AlternateAirport     string `json:"alternate_airport"`      // 目的地备降机场 (optional)
	OtherInfo            string `json:"other_info"`             // 其他信息 (optional)
	ChangePart           string `json:"change_part"`            // 修改部分
}

// Validate validates the CHG struct fields
func (c *CHG) Validate() error {
	if c.Category == "" {
		return fmt.Errorf("category is required")
	}
	if c.AircraftID == "" {
		return fmt.Errorf("aircraft id is required")
	}
	if c.SSRModeAndCode == "" {
		return fmt.Errorf("ssr mode and code is required")
	}
	if c.DepartureAirport == "" {
		return fmt.Errorf("departure airport is required")
	}
	if c.DepartureTime == "" {
		return fmt.Errorf("departure time is required")
	}
	if c.ArrivalAirport == "" {
		return fmt.Errorf("arrival airport is required")
	}
	if c.ArrivalTime == "" {
		return fmt.Errorf("arrival time is required")
	}
	if c.EstimatedElapsedTime == "" {
		return fmt.Errorf("estimated elapsed time is required")
	}
	if c.ChangePart == "" {
		return fmt.Errorf("change part is required")
	}
	return nil
}
