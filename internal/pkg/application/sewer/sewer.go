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
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Distance     float64       `json:"distance"`
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

	log := logging.GetFromContext(ctx)

	m := struct {
		Pack      *senml.Pack `json:"pack,omitempty"`
		Timestamp time.Time   `json:"timestamp"`
	}{}
	err = json.Unmarshal(itm.Body(), &m)
	if err != nil {
		return changed, err
	}

	if m.Pack == nil {
		return false, nil
	}

	log.Debug("handle distance in sewer function")

	if s.Sewer == nil {
		if t, err := tc.FindByID(ctx, s.ID); err == nil {
			s.Sewer = &t
		}
	}

	eq := func(a, b float64) bool {
		return math.Abs(a-b) <= 0.0001
	}

	sensorValue, recOk := m.Pack.GetRecord(senml.FindByName("5700"))
	if recOk {
		distance, valueOk := sensorValue.GetValue()
		if valueOk {
			log.Debug(fmt.Sprintf("current distance %f, incoming distance %f", s.Distance, distance))

			if !eq(s.Distance, distance) {
				s.Distance = distance
			}
			changed = true
		}

		ts, timeOk := sensorValue.GetTime()
		if timeOk {
			if ts.After(s.DateObserved) {
				log.Debug("set time from record")

				s.DateObserved = ts
				changed = true
			}
		}
	}

	if s.DateObserved.IsZero() {
		s.DateObserved = time.Now().UTC()
		changed = true
	}

	return changed, nil
}
