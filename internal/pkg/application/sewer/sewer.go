package sewer

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

var SewerFactory = func(id, tenant string) *Sewer {
	return &Sewer{
		ID:     id,
		Type:   "Sewer",
		Tenant: tenant,
	}
}

type Sewer struct {
	ID               string        `json:"id"`
	Type             string        `json:"type"`
	DeviceID         *string       `json:"deviceID,omitempty"`
	Level            float64       `json:"level"`
	LevelObserved    *time.Time    `json:"levelObserved"`
	Distance         *float64      `json:"distance,omitempty"`
	DistanceObserved *time.Time    `json:"distanceObserved,omitempty"`
	Percent          *float64      `json:"percent,omitempty"`
	PercentObserved  *time.Time    `json:"percentObserved,omitempty"`
	DateObserved     time.Time     `json:"dateObserved"`
	Tenant           string        `json:"tenant"`
	Sewer            *things.Thing `json:"sewer,omitempty"`
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

	log := logging.GetFromContext(ctx)

	eq := func(a, b *float64) bool {
		if a != nil && b != nil {
			return math.Abs(*a-*b) <= 0.0001
		}
		return false
	}

	m := struct {
		ID       string      `json:"id,omitempty"`
		DeviceID *string     `json:"deviceID,omitempty"`
		Tenant   *string     `json:"tenant,omitempty"`
		Pack     *senml.Pack `json:"pack,omitempty"`
		Level    *struct {
			Current float64  `json:"current"`
			Percent *float64 `json:"percent,omitempty"`
		} `json:"level,omitempty"`
		Timestamp time.Time `json:"timestamp"`
	}{}
	err = json.Unmarshal(itm.Body(), &m)
	if err != nil {
		return changed, err
	}

	log.Debug(string(itm.Body()))

	if m.Pack == nil && m.Level == nil {
		return false, nil
	}

	if s.Sewer == nil {
		if t, err := tc.FindByID(ctx, s.ID); err == nil {
			s.Sewer = &t
		}
	}

	// 1:1 device:sewer is a limitation
	if m.DeviceID != nil && s.DeviceID == nil {
		s.DeviceID = m.DeviceID
	}

	if m.Pack != nil {
		sensorValue, recOk := m.Pack.GetRecord(senml.FindByName("5700"))
		if recOk {

			ts, timeOk := sensorValue.GetTime()
			if timeOk {
				if ts.After(s.DateObserved) {
					s.DateObserved = ts
					changed = true
				}
			}

			distance, valueOk := sensorValue.GetValue()
			if valueOk {
				if !eq(s.Distance, &distance) {
					s.Distance = &distance
					if timeOk {
						s.DistanceObserved = &ts
					} else {
						now := time.Now().UTC()
						s.DistanceObserved = &now
					}
					changed = true
				}
			}

			if urn, ok := m.Pack.GetStringValue(senml.FindByName("0")); ok {
				log.Debug(fmt.Sprintf("sewer received %s measurement with value %f and changed is %t", urn, distance, changed))
			}
		}
	}

	if m.Level != nil {
		if !eq(&s.Level, &m.Level.Current) {
			s.Level = m.Level.Current
			s.LevelObserved = &m.Timestamp
			changed = true
		}

		if m.Level.Percent != nil {
			if !eq(s.Percent, m.Level.Percent) {
				s.Percent = m.Level.Percent
				s.PercentObserved = &m.Timestamp
				changed = true
			}
		}

		log.Debug(fmt.Sprintf("sewer received level with value %f and change is %t", m.Level.Current, changed))

		if m.Timestamp.After(s.DateObserved) {
			s.DateObserved = m.Timestamp
			changed = true
		}
	}

	if s.DateObserved.IsZero() {
		log.Debug("dateObserved is zero, set to Now()")
		s.DateObserved = time.Now().UTC()
		changed = true
	}

	if !changed {
		log.Debug("no change in sewer, will force change to true")
		changed = true
	}

	return changed, nil
}
