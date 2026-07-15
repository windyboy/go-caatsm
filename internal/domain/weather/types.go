package weather

import "time"

// ReportType represents the type of weather report
type ReportType string

const (
	ReportTypeMETAR ReportType = "METAR"
	ReportTypeSPECI ReportType = "SPECI"
	ReportTypeTAF   ReportType = "TAF"
)

// WeatherMessage is the common interface for all weather messages
type WeatherMessage interface {
	Type() ReportType
	Station() string
	IssueTime() time.Time
	RawText() string
}

// Metar represents a METAR or SPECI weather report
type Metar struct {
	ReportType ReportType   `json:"type"`
	StationID  string       `json:"station"`
	IssueTimeVal time.Time    `json:"issue_time"`
	ObsTime    time.Time    `json:"obs_time,omitempty"`
	RawTextVal    string       `json:"raw_text"`

	// Core elements
	Wind        *Wind        `json:"wind,omitempty"`
	Visibility  *Visibility  `json:"visibility,omitempty"`
	Clouds      []Cloud      `json:"clouds,omitempty"`
	Temperature *Temperature `json:"temperature,omitempty"`
	Dewpoint    *Temperature `json:"dewpoint,omitempty"`
	Altimeter   *Altimeter   `json:"altimeter,omitempty"`
	Phenomena   []Phenomenon `json:"phenomena,omitempty"`

	// Optional fields
	Modifier string   `json:"modifier,omitempty"` // AUTO, COR
	Remarks  string   `json:"remarks,omitempty"`
	Warnings []string `json:"warnings,omitempty"` // Unrecognized tokens
}

// Type returns the report type
func (m *Metar) Type() ReportType {
	return m.ReportType
}

// Station returns the station identifier
func (m *Metar) Station() string {
	return m.StationID
}

// IssueTime returns the issue time
func (m *Metar) IssueTime() time.Time {
	return m.IssueTimeVal
}

// RawText returns the raw text
func (m *Metar) RawText() string {
	return m.RawTextVal
}

// Taf represents a TAF (Terminal Aerodrome Forecast) weather report
type Taf struct {
	ReportType ReportType  `json:"type"`
	StationID  string      `json:"station"`
	IssueTimeVal  time.Time   `json:"issue_time"`
	ValidFrom  time.Time   `json:"valid_from"`
	ValidTo    time.Time   `json:"valid_to"`
	RawTextVal    string      `json:"raw_text"`

	Periods  []TafPeriod `json:"periods"` // FM, TEMPO, BECMG segments
	Remarks  string       `json:"remarks,omitempty"`
	Warnings []string     `json:"warnings,omitempty"`
}

// Type returns the report type
func (t *Taf) Type() ReportType {
	return t.ReportType
}

// Station returns the station identifier
func (t *Taf) Station() string {
	return t.StationID
}

// IssueTime returns the issue time
func (t *Taf) IssueTime() time.Time {
	return t.IssueTimeVal
}

// RawText returns the raw text
func (t *Taf) RawText() string {
	return t.RawTextVal
}

