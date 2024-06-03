package sewer

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
)

var SewerFactory = func(id, tenant string) *Sewer {
	return &Sewer{
		ID:     id,
		Type:   "Sewer",
		Tenant: tenant,
	}
}

type Sewer struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Distance     float64       `json:"distance"`
	Level        float64       `json:"level"`
	DateObserved time.Time     `json:"dateObserved"`
	Tenant       string        `json:"tenant"`
	Sewer        *things.Thing `json:"sewer,omitempty"`
}

func (s Sewer) TopicName() string {
	return "cip-function.updated"
}

func (s Sewer) ContentType() string {
	return "application/vnd.diwise.sewer+json"
}

func (s Sewer) Body() []byte {
	b, _ := json.Marshal(s)
	return b
}

func (s *Sewer) Handle(ctx context.Context, itm messaging.IncomingTopicMessage, tc things.Client) (bool, error) {
	var err error
	changed := false

	eq := func(a, b float64) bool {
		return math.Abs(a-b) <= 0.0001
	}

	m := struct {
		ID     string      `json:"id,omitempty"`
		Tenant *string     `json:"tenant,omitempty"`
		Pack   *senml.Pack `json:"pack,omitempty"`
		Level  *struct {
			Current float64  `json:"current"`
			Percent *float64 `json:"percent,omitempty"`
		} `json:"level,omitempty"`
		Timestamp time.Time `json:"timestamp"`
	}{}
	err = json.Unmarshal(itm.Body(), &m)
	if err != nil {
		return changed, err
	}

	if m.Pack == nil && m.Level == nil {
		return false, nil
	}

	if s.Sewer == nil {
		if t, err := tc.FindByID(ctx, s.ID); err == nil {
			s.Sewer = &t
		}
	}

	if m.Pack != nil {
		sensorValue, recOk := m.Pack.GetRecord(senml.FindByName("5700"))
		if recOk {
			distance, valueOk := sensorValue.GetValue()
			if valueOk {
				if !eq(s.Distance, distance) {
					s.Distance = distance
					changed = true
				}
			}

			ts, timeOk := sensorValue.GetTime()
			if timeOk {
				if ts.After(s.DateObserved) {
					s.DateObserved = ts
					changed = true
				}
			}
		}

		if s.DateObserved.IsZero() {
			s.DateObserved = time.Now().UTC()
			changed = true
		}
	}

	if m.Level != nil {
		if !eq(s.Distance, m.Level.Current) {
			s.Distance = m.Level.Current
			changed = true
		}
		if m.Level.Percent != nil {
			if !eq(s.Level, *m.Level.Percent) {
				s.Level = *m.Level.Percent
				changed = true
			}
		}
		if changed && m.Timestamp.After(s.DateObserved) {
			s.DateObserved = m.Timestamp			
		}
	}

	return changed, nil
}
