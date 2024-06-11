package entities

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
}

// func (r *Reading) Key() string {
// 	return fmt.Sprintf("%s.%s.%s", r.Event, r.MeterId, r.Subject)
// }
