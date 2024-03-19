package sewagepumpingstation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/messaging-golang/pkg/messaging"
)

var SewagePumpingStationFactory = func(id, tenant string) *SewagePumpingStation {
	return &SewagePumpingStation{
		ID:     id,
		Type:   "SewagePumpingStation",
		Tenant: tenant,
	}
}

type SewagePumpingStation struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	State      bool       `json:"state"`
	Tenant     string     `json:"tenant"`
	ObservedAt *time.Time `json:"observedAt"`
}

func (sp SewagePumpingStation) Body() []byte {

	bytes, err := json.Marshal(sp)
	if err != nil {
		return []byte{}
	}

	return bytes
}

func (sp SewagePumpingStation) TopicName() string {
	return "cip-function.updated"
}

func (sp SewagePumpingStation) ContentType() string {
	return "application/vnd.diwise.sewagepumpingstation+json"
}

func (sp *SewagePumpingStation) Handle(ctx context.Context, itm messaging.IncomingTopicMessage) (bool, error) {

	m := struct {
		ID           string    `json:"id"`
		Tenant       string    `json:"tenant,omitempty"`
		Timestamp    time.Time `json:"timestamp,omitempty"`
		DigitalInput struct {
			Timestamp string `json:"timestamp"`
			State     bool   `json:"state"`
		}
	}{}

	json.Unmarshal(itm.Body(), &m)

	changed := sp.State != m.DigitalInput.State || sp.ObservedAt != &m.Timestamp

	if sp.Tenant == "" {
		sp.Tenant = m.Tenant
	}

	sp.State = m.DigitalInput.State
	sp.ObservedAt = &m.Timestamp

	return changed, nil
}
