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

const FunctionName string = "sumppump"

type SumpPump interface {
	Handle(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error
}

type sp struct {
}

func New() SumpPump {
	sp := &sp{}

	return sp
}

type SumpPumpObserved struct {
	ID           string     `json:"id"`
	AlertID      string     `json:"alertId,omitempty"`
	State        bool       `json:"state"`
	StartTime    *time.Time `json:"startTime,omitempty"`
	EndTime      *time.Time `json:"endTime,omitempty"`
	LastObserved *time.Time `json:"lastObserved"`
}

func (sp *sp) Handle(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error {
	//generate ID
	id := fmt.Sprintf("SumpPumpObserved:%s", msg.ID)

	//check if it already exists in database
	exists := storage.Exists(ctx, id)
	if !exists {
		time := time.Now().UTC()
		err := storage.Create(ctx, id, SumpPumpObserved{
			ID:           id,
			State:        msg.Stopwatch.State,
			LastObserved: &time,
		})
		if err != nil {
			return err
		}

		if msg.Stopwatch.State {
			alertID := fmt.Sprintf("Alert:alertID:%s", id) //TODO: better ID generation
			err = storage.Create(ctx, fmt.Sprintf("%s:alert:", id), Alert{
				ID:        alertID,
				State:     msg.Stopwatch.State,
				StartTime: &msg.Stopwatch.StartTime,
			})
			if err != nil {
				return err
			}
		}
	} else if exists {
		spo := SumpPumpObserved{}

		spo, err := database.Get[SumpPumpObserved](ctx, storage, id)
		if err != nil {
			return err
		}

		//if it already does and state has not changed, update dateModified

		//if it already exists, and state has now changed create a new pumpbrunn/alert
		if spo.State != msg.Stopwatch.State {
			if spo.AlertID == "" {
				alert := Alert{
					ID:        "generateAnAlertID",
					Owner:     spo.ID,
					State:     msg.Stopwatch.State,
					StartTime: &msg.Stopwatch.StartTime,
				}
				storage.Create(ctx, id, alert)

				//TODO: send created alert on queue

				spo.AlertID = alert.ID

				storage.Update(ctx, id, spo)
			} else {
				//update existing alert with new timestamp and/or endTime
				//spo.LastObserved = msg.LastObserved or something like that.
				storage.Update(ctx, id, spo)
			}
		} else {
			//if state is the same, but there IS an active alert, update lastObserved
			//post message to cip-functions.updated
		}
	}

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
