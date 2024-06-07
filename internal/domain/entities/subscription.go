package entities

type Meter struct {
	Slug          string            `json:"slug"`
	Description   string            `json:"description"`
	EventType     string            `json:"eventType"`
	Aggregation   string            `json:"aggregation"`
	ValueProperty string            `json:"valueProperty"`
	GroupBy       map[string]string `json:"groupBy"`
}

type Event struct {
	SpecVersion string         `json:"specVersion"`
	Type        string         `json:"type"`
	Id          string         `json:"id"`
	Time        string         `json:"time"`
	Source      string         `json:"source"`
	Subject     string         `json:"subject"`
	Data        map[string]any `json:"data"`
}
