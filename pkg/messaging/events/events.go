package events

import (
	"errors"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/topics"
)

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type MessageReceived struct {
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

func (m *MessageReceived) ContentType() string {
	return "application/json"
}

func (m MessageReceived) FunctionID() string {
	return m.ID_
}

type EventDecoratorFunc func(m *MessageAccepted)
type MessageAccepted struct {
	Function  string `json:"functionID"`
	Timestamp string `json:"timestamp"`
}

func NewMessageAccepted(functionID string, decorators ...EventDecoratorFunc) *MessageAccepted {
	m := &MessageAccepted{
		Function:  functionID,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
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
	if m.Function == "" {
		return errors.New("function id is missing")
	}

	if m.Timestamp == "" {
		return errors.New("timestamp is mising")
	}

	//add check for function names, such as levels, counters, stopwatch etc. If none exist, return error.

	return nil
}
