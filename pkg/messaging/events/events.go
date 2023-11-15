package events

import (
	"errors"

	"github.com/diwise/cip-functions/pkg/messaging/topics"
)

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type FunctionUpdated struct {
	ID_      string    `json:"id"`
	Name_    string    `json:"name"`
	Type     string    `json:"type"`
	SubType  string    `json:"subtype"`
	Location *location `json:"location,omitempty"`
	Tenant   string    `json:"tenant,omitempty"`
	Source   string    `json:"source,omitempty"`

	/*Counter      counters.Counter            `json:"counter,omitempty"`
	Level        levels.Level                `json:"level,omitempty"`
	Presence     presences.Presence          `json:"presence,omitempty"`
	Timer        timers.Timer                `json:"timer,omitempty"`
	WaterQuality waterqualities.WaterQuality `json:"waterquality,omitempty"`
	Building     buildings.Building          `json:"building,omitempty"`
	AirQuality   airquality.AirQuality       `json:"AirQuality,omitempty"`*/
	Stopwatch Stopwatch `json:"Stopwatch,omitempty"`
}

func (m *FunctionUpdated) ID() string {
	return m.ID_
}

type EventDecoratorFunc func(m *MessageAccepted)

type MessageAccepted struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func NewMessageAccepted(functionID string, msg FunctionUpdated, decorators ...EventDecoratorFunc) *MessageAccepted {
	m := &MessageAccepted{
		ID:   msg.ID_,
		Type: msg.Type,
	}

	for _, d := range decorators {
		d(m)
	}

	return m
}

func (m *MessageAccepted) ContentType() string {
	return "application/json"
}

func (m *MessageAccepted) TopicName() string {
	return topics.MessageAccepted
}

func (m *MessageAccepted) Error() error {
	if m.ID == "" {
		return errors.New("function id is missing")
	}

	//add check for function names, such as levels, counters, stopwatch etc. If none exist, return error.

	return nil
}
