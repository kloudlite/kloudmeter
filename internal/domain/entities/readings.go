package entities

import "time"

type DurationData struct {
	Total          float64   `json:"total"`
	Unit           float64   `json:"unit"`
	LastCalculated time.Time `json:"lastCalculated"`
}

type Reading struct {
	Event   string `json:"event"`
	MeterId string `json:"meterId"`
	Subject string `json:"subject"`
	Segment string `json:"segment,omitempty"`

	Type AggType `json:"type"`

	Count int     `json:"count,omitempty"`
	Sum   float64 `json:"sum,omitempty"`
	Avg   float64 `json:"avg,omitempty"`
	Max   float64 `json:"max,omitempty"`
	Min   float64 `json:"min,omitempty"`

	Func string `json:"func,omitempty"`

	Unique map[string]int `json:"unique,omitempty"`

	DurationData DurationData `json:"durationData,omitempty"`
}

// func (r *Reading) Key() string {
// 	return fmt.Sprintf("%s.%s.%s", r.Event, r.MeterId, r.Subject)
// }
