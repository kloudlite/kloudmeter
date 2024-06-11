package entities

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/kloudlite/kloudmeter/pkg/egob"
)

type Event struct {
	Id   string `json:"id"`
	Time string `json:"time"`

	EventType string         `json:"eventType"`
	Subject   string         `json:"subject"`
	Data      map[string]any `json:"data"`
}

func (e *Event) Key() string {
	return fmt.Sprintf("%s.%s.%s", e.EventType, e.Subject, e.Id)
}

func (e *Event) ParseBytes(b []byte) error {
	return egob.Unmarshal(b, e)
}

func (e *Event) ToJson() ([]byte, error) {
	return egob.Marshal(e)
}

func (m *Event) IsValid() error {
	if m.EventType == "" {
		return errors.New("eventType is required")
	}

	if m.Id == "" {
		return errors.New("id is required")
	}

	if m.Subject == "" {
		return errors.New("subject is required")
	}

	// no space or special chars
	subjectRegex := regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	if !subjectRegex.MatchString(m.Subject) {
		return errors.New("subject can only contain alphanumeric characters, dashes and underscores")
	}

	return nil
}
