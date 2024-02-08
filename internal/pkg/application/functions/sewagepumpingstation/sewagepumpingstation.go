package sewagepumpingstation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/functions/options"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
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

func (sp *IncomingSewagePumpingStation) Handle(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error {

	log := logging.GetFromContext(ctx)

	if msg.Type != "state" {
		log.Info("invalid function type", "id", msg.ID, "type", msg.Type, "sub_type", msg.SubType)
		return nil
	}

	id := msg.ID

	exists := storage.Exists(ctx, id)
	if !exists {
		spo := SewagePumpingStation{
			ID:         id,
			State:      msg.State.State_,
			Tenant:     msg.Tenant,
			ObservedAt: &msg.Timestamp,
		}

		err := storage.Create(ctx, id, spo)
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
		spo, err := database.Get[SewagePumpingStation](ctx, storage, id)
		if err != nil {
			log.Error("could not retrieve sewagepumpingstation from storage", "id", id, "msg", err)
			return err
		}

		spo.State = msg.State.State_
		spo.ObservedAt = &msg.Timestamp

		storage.Update(ctx, id, spo)

		err = msgCtx.PublishOnTopic(ctx, spo)
		if err != nil {
			return fmt.Errorf("failed to publish updated sewagepumpingstation message: %s", err)
		}
		log.Info("published message", "id", spo.ID, "topic", spo.TopicName())

	}

	return nil
}
