package sewagepumpingstation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/functions/options"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/cip-functions/pkg/messaging/topics"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const FunctionName string = "sewagepumpingstation"

type IncomingSewagePumpingStation struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	State      bool       `json:"state"`
	Tenant     string     `json:"tenant"`
	ObservedAt *time.Time `json:"observedAt"`
}

type SewagePumpingStation struct {
	ID         string     `json:"id"`
	State      bool       `json:"state"`
	Tenant     string     `json:"tenant"`
	ObservedAt *time.Time `json:"observedAt"`
}

func New() IncomingSewagePumpingStation {
	return IncomingSewagePumpingStation{}
}

func (sp SewagePumpingStation) Body() []byte {

	bytes, err := json.Marshal(sp)
	if err != nil {
		return []byte{}
	}

	return bytes
}

func (sp SewagePumpingStation) TopicName() string {
	return topics.CipFunctionUpdated
}

func (sp SewagePumpingStation) ContentType() string {
	return "application/vnd+diwise.sewagepumpingstation+json"
}

func (sp *IncomingSewagePumpingStation) Handle(ctx context.Context, msg *events.FunctionUpdated, store storage.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error {

	log := logging.GetFromContext(ctx)

	if msg.Type != "digitalinput" {
		log.Info("invalid function type", "id", msg.ID, "type", msg.Type, "sub_type", msg.SubType)
		return nil
	}

	id := msg.ID

	timestamp, err := time.Parse(time.RFC3339, msg.DigitalInput.Timestamp)
	if err != nil {
		log.Error("failed to parse time from state", "id", id, "msg", err)
	}

	exists := store.Exists(ctx, id, "SewagePumpingStation")
	if !exists {
		spo := SewagePumpingStation{
			ID:         id,
			State:      msg.DigitalInput.State,
			Tenant:     msg.Tenant,
			ObservedAt: &timestamp,
		}

		err := store.Create(ctx, id, spo)
		if err != nil {
			log.Error("failed to create new sewagepumpingstation in storage")
			return err
		}

		err = msgCtx.PublishOnTopic(ctx, spo)
		if err != nil {
			log.Error("failed to publish new sewagepumpingstation message")
			return err
		}

		log.Info("published message", "id", spo.ID, "topic", spo.TopicName())

	} else {
		spo, err := storage.Get[SewagePumpingStation](ctx, store, id)
		if err != nil {
			log.Error("could not retrieve sewagepumpingstation from storage", "id", id, "msg", err)
			return err
		}

		spo.State = msg.DigitalInput.State

		spo.ObservedAt = &timestamp

		store.Update(ctx, id, spo)

		err = msgCtx.PublishOnTopic(ctx, spo)
		if err != nil {
			return fmt.Errorf("failed to publish updated sewagepumpingstation message: %s", err)
		}
		log.Info("published message", "id", spo.ID, "topic", spo.TopicName())

	}

	return nil
}
