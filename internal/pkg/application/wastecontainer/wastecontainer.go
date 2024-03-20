package wastecontainer

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

var WasteContainerFactory = func(id, tenant string) *WasteContainer {
	return &WasteContainer{
		ID:     id,
		Type:   "WasteContainer",
		Tenant: tenant,
	}
}

type WasteContainer struct {
	ID             string        `json:"id"`
	Type           string        `json:"type"`
	Level          float64       `json:"level"`
	Percent        float64       `json:"percent"`
	Temperature    float64       `json:"temperature"`
	DateObserved   time.Time     `json:"dateObserved"`
	Tenant         string        `json:"tenant"`
	WasteContainer *things.Thing `json:"wastecontainer,omitempty"`
}

func (wc WasteContainer) TopicName() string {
	return "cip-function.updated"
}

func (wc WasteContainer) ContentType() string {
	return "application/vnd.diwise.wastecontainer+json"
}

func (wc WasteContainer) Body() []byte {
	b, _ := json.Marshal(wc)
	return b
}

func (wc *WasteContainer) Handle(ctx context.Context, itm messaging.IncomingTopicMessage, tc things.Client) (bool, error) {
	var err error
	changed := false

	log := logging.GetFromContext(ctx)

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

	if wc.WasteContainer == nil {
		if t, err := tc.FindByID(ctx, wc.ID); err == nil {
			wc.WasteContainer = &t
		}
	}

	if m.Level != nil {
		wc.Level = m.Level.Current

		if m.Level.Percent != nil {
			wc.Percent = math.Round(*m.Level.Percent)
		}

		ts := time.Now().UTC()
		if !m.Timestamp.IsZero() {
			ts = m.Timestamp
		}

		wc.DateObserved = ts
		changed = true
	}

	if m.Pack == nil {
		log.Debug(fmt.Sprintf("message contains no pack, is changed: %t", changed))
		return changed, nil
	}

	sensorValue, recOk := m.Pack.GetRecord(senml.FindByName("5700"))
	if recOk {
		t, valueOk := sensorValue.GetValue()
		if valueOk {
			wc.Temperature = t
			changed = true
		}
		ts, timeOk := sensorValue.GetTime()
		if timeOk {
			if ts.After(wc.DateObserved) {
				wc.DateObserved = ts
				changed = true
			}
		}
	}

	if wc.DateObserved.IsZero() {
		wc.DateObserved = time.Now().UTC()
		changed = true
	}

	return changed, nil
}
