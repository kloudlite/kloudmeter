package entities

import (
	"crypto/md5"
	"errors"
	"fmt"
)

type AggType string

const (
	AggTypeCount  AggType = "count"
	AggTypeSum    AggType = "sum"
	AggTypeAvg    AggType = "avg"
	AggTypeMax    AggType = "max"
	AggTypeMin    AggType = "min"
	AggTypeRange  AggType = "range"
	AggTypeFunc   AggType = "func"
	AggTypeUnique AggType = "unique"
)

type Meter struct {
	Id          string `json:"id"`
	Description string `json:"description"`

	EventType string `json:"eventType"`

	Aggregation   AggType           `json:"aggregation"`
	ValueProperty string            `json:"valueProperty"`
	GroupBy       map[string]string `json:"groupBy"`
}

func (m *Meter) Key() string {
	return fmt.Sprintf("%s.%s.%s", m.EventType, m.Aggregation, m.Id)
}

func (m *Meter) Hash() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(m.Key())))
}

func (m *Meter) IsValid() error {
	if m.Id == "" {
		return errors.New("id is required")
	}

	if m.EventType == "" {
		return errors.New("eventType is required")
	}

	if m.Aggregation == "" {
		return errors.New("aggregation is required")
	}

	if m.ValueProperty == "" {
		return errors.New("valueProperty is required")
	}

	return nil
}
