package weather

import (
	"errors"
	"time"
)

var (
	// ErrInvalidFormat indicates an invalid weather report format.
	ErrInvalidFormat = errors.New("invalid weather report format")

	// ErrUnsupportedToken indicates an unsupported token in the report.
	ErrUnsupportedToken = errors.New("unsupported token")

	// ErrMissingStation indicates missing station identifier.
	ErrMissingStation = errors.New("missing station identifier")

	// ErrMissingTime indicates missing time information.
	ErrMissingTime = errors.New("missing time information")
)

// ReportType represents the type of weather report.
type ReportType string

const (
	ReportTypeMETAR ReportType = "METAR"
	ReportTypeSPECI ReportType = "SPECI"
	ReportTypeTAF   ReportType = "TAF"
)

// Message is the common interface for all parsed weather reports.
type Message interface {
	Type() ReportType
	Station() string
	IssueTime() time.Time
	RawText() string
}

// Metar represents a METAR or SPECI weather report.
type Metar struct {
	ReportType   ReportType `json:"type"`
	StationID    string     `json:"station"`
	IssueTimeVal time.Time  `json:"issue_time"`
	ObsTime      time.Time  `json:"obs_time,omitempty"`
	RawTextVal   string     `json:"raw_text"`

	Wind        *Wind        `json:"wind,omitempty"`
	Visibility  *Visibility  `json:"visibility,omitempty"`
	Clouds      []Cloud      `json:"clouds,omitempty"`
	Temperature *Temperature `json:"temperature,omitempty"`
	Dewpoint    *Temperature `json:"dewpoint,omitempty"`
	Altimeter   *Altimeter   `json:"altimeter,omitempty"`
	Phenomena   []Phenomenon `json:"phenomena,omitempty"`

	Modifier string   `json:"modifier,omitempty"`
	Remarks  string   `json:"remarks,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func (m *Metar) Type() ReportType     { return m.ReportType }
func (m *Metar) Station() string      { return m.StationID }
func (m *Metar) IssueTime() time.Time { return m.IssueTimeVal }
func (m *Metar) RawText() string      { return m.RawTextVal }

// Taf represents a TAF (Terminal Aerodrome Forecast) weather report.
type Taf struct {
	ReportType   ReportType `json:"type"`
	StationID    string     `json:"station"`
	IssueTimeVal time.Time  `json:"issue_time"`
	ValidFrom    time.Time  `json:"valid_from"`
	ValidTo      time.Time  `json:"valid_to"`
	RawTextVal   string     `json:"raw_text"`

	Periods  []TafPeriod `json:"periods"`
	Modifier string      `json:"modifier,omitempty"`
	Remarks  string      `json:"remarks,omitempty"`
	Warnings []string    `json:"warnings,omitempty"`
}

func (t *Taf) Type() ReportType     { return t.ReportType }
func (t *Taf) Station() string      { return t.StationID }
func (t *Taf) IssueTime() time.Time { return t.IssueTimeVal }
func (t *Taf) RawText() string      { return t.RawTextVal }

// Wind represents wind information.
type Wind struct {
	Direction    int    `json:"direction"`
	Speed        int    `json:"speed"`
	Gust         int    `json:"gust,omitempty"`
	Variable     bool   `json:"variable,omitempty"`
	VariableFrom int    `json:"variable_from,omitempty"`
	VariableTo   int    `json:"variable_to,omitempty"`
	Unit         string `json:"unit"`
}

// Visibility represents visibility information.
type Visibility struct {
	Distance  float64 `json:"distance"`
	Unit      string  `json:"unit"`
	Direction string  `json:"direction,omitempty"`
	Modifier  string  `json:"modifier,omitempty"`
}

// Cloud represents cloud layer information.
type Cloud struct {
	Type     string `json:"type"`
	Altitude int    `json:"altitude"`
	Modifier string `json:"modifier,omitempty"`
}

// Temperature represents temperature or dewpoint.
type Temperature struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// Altimeter represents altimeter pressure setting.
type Altimeter struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// Phenomenon represents weather phenomenon.
type Phenomenon struct {
	Intensity  string `json:"intensity,omitempty"`
	Descriptor string `json:"descriptor,omitempty"`
	Weather    string `json:"weather"`
}

// TafPeriod represents a TAF period (FM, TEMPO, BECMG, or MAIN forecast).
type TafPeriod struct {
	Type        string       `json:"type"`
	ValidFrom   time.Time    `json:"valid_from,omitempty"`
	ValidTo     time.Time    `json:"valid_to,omitempty"`
	Wind        *Wind        `json:"wind,omitempty"`
	Visibility  *Visibility  `json:"visibility,omitempty"`
	Clouds      []Cloud      `json:"clouds,omitempty"`
	Phenomena   []Phenomenon `json:"phenomena,omitempty"`
	Probability int          `json:"probability,omitempty"`
}
