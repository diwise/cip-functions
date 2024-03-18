package wastecontainer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
)

var WasteContainerFactory = func(id, tenant string) *WasteContainer {
	return &WasteContainer{
		ID:     id,
		Type:   "WasteContainer",
		Tenant: tenant,
	}
}

type WasteContainer struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Level        float64   `json:"level"`
	Temperature  float64   `json:"temperature"`
	DateObserved time.Time `json:"dateObserved"`
	Tenant       string    `json:"tenant"`
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

func (wc *WasteContainer) Handle(ctx context.Context, itm messaging.IncomingTopicMessage) (bool, error) {
	var err error
	changed := false

	m := struct {
		ID     *string     `json:"id,omitempty"`
		Tenant *string     `json:"tenant,omitempty"`
		Pack   *senml.Pack `json:"pack,omitempty"`
		Level  *struct {
			Current float64  `json:"current"`
			Percent *float64 `json:"percent,omitempty"`
		} `json:"level,omitempty"`
	}{}

	err = json.Unmarshal(itm.Body(), &m)
	if err != nil {
		return changed, err
	}

	if m.Level != nil && m.Level.Percent != nil {
		wc.Level = *m.Level.Percent
		wc.DateObserved = time.Now().UTC()
		changed = true
	}

	if m.Pack == nil {
		return changed, nil
	}

	if t, ok := m.Pack.GetValue(senml.FindByName("5700")); ok {
		wc.Temperature = t
		changed = true
	}

	if ts, ok := m.Pack.GetTime(senml.FindByName("5700")); ok {
		if ts.After(wc.DateObserved) {
			wc.DateObserved = ts
			changed = true
		}
	}

	if wc.DateObserved.IsZero() {
		wc.DateObserved = time.Now().UTC()
		changed = true
	}

	return changed, nil
}
