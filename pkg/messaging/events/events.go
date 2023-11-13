package events

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/diwise/iot-core/pkg/messaging/topics"
	"github.com/farshidtz/senml/v2"
)

type MessageReceived struct {
	Device    string     `json:"deviceID"`
	Pack      senml.Pack `json:"pack"`
	Timestamp string     `json:"timestamp"`
}

func (m *MessageReceived) ContentType() string {
	return "application/json"
}

func (m MessageReceived) DeviceID() string {
	if m.Pack[0].Name == "0" {
		return m.Pack[0].StringValue
	}

	return ""
}

type EventDecoratorFunc func(m *MessageAccepted)
type MessageAccepted struct {
	Sensor    string     `json:"sensorID"`
	Pack      senml.Pack `json:"pack"`
	Timestamp string     `json:"timestamp"`
}

func NewMessageAccepted(sensorID string, pack senml.Pack, decorators ...EventDecoratorFunc) *MessageAccepted {
	m := &MessageAccepted{
		Sensor:    sensorID,
		Pack:      pack,
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
	if m.Sensor == "" {
		return errors.New("sensor id is missing")
	}

	if m.Timestamp == "" {
		return errors.New("timestamp is mising")
	}

	if len(m.Pack) == 0 {
		return errors.New("pack is empty")
	}

	return nil
}

func Rec(n, vs string, v *float64, vb *bool, t float64, sum *float64) EventDecoratorFunc {
	return func(m *MessageAccepted) {
		for _, r := range m.Pack {
			if strings.EqualFold(r.Name, n) {
				r.StringValue = vs
				r.Value = v
				r.BoolValue = vb
				r.Time = t
				r.Sum = sum
				return
			}
		}

		rec := senml.Record{
			Name:        n,
			StringValue: vs,
			Value:       v,
			BoolValue:   vb,
			Time:        t,
			Sum:         sum,
		}

		m.Pack = append(m.Pack, rec)
	}
}

func Lat(t float64) EventDecoratorFunc {
	return func(m *MessageAccepted) {
		for _, r := range m.Pack {
			if r.Unit == senml.UnitLat {
				r.Value = &t
				return
			}
		}

		lat := &senml.Record{
			Unit:  senml.UnitLat,
			Value: &t,
		}

		m.Pack = append(m.Pack, *lat)
	}
}

func Lon(t float64) EventDecoratorFunc {
	return func(m *MessageAccepted) {
		for _, r := range m.Pack {
			if r.Unit == senml.UnitLon {
				r.Value = &t
				return
			}
		}

		lat := &senml.Record{
			Unit:  senml.UnitLon,
			Value: &t,
		}

		m.Pack = append(m.Pack, *lat)
	}
}

func Environment(e string) EventDecoratorFunc {
	if strings.EqualFold(e, "") {
		return func(m *MessageAccepted) {}
	}
	return Rec("env", e, nil, nil, 0, nil)
}

func Source(s string) EventDecoratorFunc {
	if strings.EqualFold(s, "") {
		return func(m *MessageAccepted) {}
	}
	return Rec("source", s, nil, nil, 0, nil)
}

func Tenant(t string) EventDecoratorFunc {
	if strings.EqualFold(t, "") {
		t = "default"
	}
	return Rec("tenant", t, nil, nil, 0, nil)
}

func (m MessageAccepted) Latitude() float64 {
	for _, r := range m.Pack {
		if r.Unit == senml.UnitLat {
			return *r.Value
		}
	}
	return 0
}

func (m MessageAccepted) Longitude() float64 {
	for _, r := range m.Pack {
		if r.Unit == senml.UnitLon {
			return *r.Value
		}
	}
	return 0
}

func (m MessageAccepted) HasLocation() bool {
	return m.Latitude() != 0 || m.Longitude() != 0
}

func (m MessageAccepted) GetFloat64(name string) (float64, bool) {
	for _, r := range m.Pack {
		if strings.EqualFold(r.Name, name) {
			if r.Value != nil {
				return *r.Value, true
			}
			return 0, false
		}
	}
	return 0, false
}

func (m MessageAccepted) GetString(name string) (string, bool) {
	for _, r := range m.Pack {
		if strings.EqualFold(r.Name, name) {
			return r.StringValue, true
		}
	}
	return "", false
}

func (m MessageAccepted) GetBool(name string) (bool, bool) {
	for _, r := range m.Pack {
		if strings.EqualFold(r.Name, name) {
			if r.BoolValue != nil {
				return *r.BoolValue, true
			}
			return false, false
		}
	}
	return false, false
}

func (m MessageAccepted) GetTime(name string) (float64, bool) {
	for _, r := range m.Pack {
		if strings.EqualFold(r.Name, name) {
			if r.Time != 0 {
				return r.Time, true
			}
			return 0, false
		}
	}
	return 0, false
}

func (m MessageAccepted) GetRecord(name string) (senml.Record, bool) {
	for _, r := range m.Pack {
		if strings.EqualFold(r.Name, name) {
			return r, true
		}
	}
	return senml.Record{}, false
}

func (m MessageAccepted) GetTimeForRec(name string) (time.Time, bool) {
	btime := m.Pack[0].BaseTime

	var now = float64(time.Now().UnixNano()) / 1000000000
	const pivot = 268435456 // rfc8428: values less than 2**28 represent time relative to the current time.

	//TODO: remove this "fix" when time is correctly used
	for _, r := range m.Pack {
		if strings.EqualFold(r.Name, name) {
			if r.Time > 0 {
				if r.Time < pivot {
					return time.Unix(int64(now+r.Time), 0).UTC(), true
				}

				return time.Unix(int64(r.Time), 0).UTC(), true
			}

			return time.Unix(int64(btime), 0).UTC(), true
		}
	}

	return time.Time{}, false
}

func (m MessageAccepted) Tenant() string {
	if s, ok := m.GetString("tenant"); ok {
		return s
	}

	return ""
}

func (m MessageAccepted) BaseName() string {
	return m.Pack[0].BaseName
}

func (m MessageAccepted) BaseNameMatches(name string) bool {
	baseName := m.BaseName()
	return (baseName == name)
}

func (m MessageAccepted) BaseTime() float64 {
	return m.Pack[0].BaseTime
}

func Get[T float64 | string | bool](m MessageAccepted, baseName string, id int) (T, bool) {
	if strings.EqualFold(m.BaseName(), baseName) {
		n := fmt.Sprint(id)
		t := *new(T)

		switch reflect.TypeOf(t).Kind() {
		case reflect.Float64:
			if v, ok := m.GetFloat64(n); ok {
				if r, ok := reflect.ValueOf(v).Interface().(T); ok {
					return r, true
				}
			}
		case reflect.Bool:
			if vb, ok := m.GetBool(n); ok {
				if r, ok := reflect.ValueOf(vb).Interface().(T); ok {
					return r, true
				}
			}
		case reflect.String:
			if vs, ok := m.GetString(n); ok {
				if r, ok := reflect.ValueOf(vs).Interface().(T); ok {
					return r, true
				}
			}
		default:
			return *new(T), false
		}
	}
	return *new(T), false
}
