package sumppump

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/functions/options"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/cip-functions/pkg/messaging/topics"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

type SumpPump interface {
	Handle(ctx context.Context, msg *events.FunctionUpdated, options ...options.Option) error
}

type sp struct {
	id    string
	state bool

	storage database.Storage
	msgCtx  messaging.MsgContext
}

func New(storage database.Storage, msgCtx messaging.MsgContext, id string) SumpPump {
	sp := &sp{
		id:    id,
		state: false,

		storage: storage,
		msgCtx:  msgCtx,
	}

	return sp
}

type SumpPumpObserved struct {
	ID           string     `json:"id"`
	AlertID      string     `json:"alertId"`
	State        bool       `json:"state"`
	StartTime    *time.Time `json:"startTime,omitempty"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	LastObserved *time.Time `json:"lastObserved"`
}

func (sp *sp) Handle(ctx context.Context, msg *events.FunctionUpdated, options ...options.Option) error {
	//generate ID
	id := fmt.Sprintf("SumpPumpObserved:%s", sp.id)

	//check if it already exists in database
	exists := sp.storage.Exists(ctx, id)
	if !exists {
		time := time.Now().UTC()
		err := sp.storage.Create(ctx, id, SumpPumpObserved{
			ID:           id,
			State:        sp.state,
			LastObserved: &time,
		})
		if err != nil {
			return err
		}
	} else if exists {
		spo, err := database.Get[SumpPumpObserved](ctx, sp.storage, id)
		if err != nil {
			return err
		}

		//if it already does and state has not changed, update dateModified

		//if it already exists, and state has now changed create a new pumpbrunn/alert
		if spo.State != msg.Stopwatch.State {
			if spo.AlertID != "" {
				sp.storage.Update(ctx, id, spo.State)
			}
			//either create new alert or close existing alert.
		}
	}

	//post message to cip-functions.updated

	return nil
}

type Alert struct {
	ID           string     `json:"id"`
	Owner        string     `json:"owner"`
	State        bool       `json:"state"`
	StartTime    *time.Time `json:"startTime"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	LastObserved *time.Time `json:"lastObserved"`
}

func (a Alert) TopicName() string {
	return topics.CipFunctionsUpdated
}

func (a Alert) ContentType() string {
	return "application/vnd+diwise.alertstarted+json"
}
