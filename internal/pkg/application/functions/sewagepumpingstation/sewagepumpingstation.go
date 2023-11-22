package sewagepumpingstation

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

const FunctionName string = "sewagepumpingstation"

type SewagePumpingStation interface {
	Handle(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error
}

type sp struct {
}

func New() SewagePumpingStation {
	sp := &sp{}

	return sp
}

type SewagePumpingStationObserved struct {
	ID             string     `json:"id"`
	ActiveAlert    string     `json:"activeAlert,omitempty"`
	PreviousAlerts []string   `json:"previousAlerts,omitempty"`
	State          bool       `json:"state"`
	StartTime      *time.Time `json:"startTime,omitempty"`
	EndTime        *time.Time `json:"endTime,omitempty"`
	LastObserved   *time.Time `json:"lastObserved"`
}

func (sp *sp) Handle(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error {
	//generate ID
	id := fmt.Sprintf("SewagePumpingStationObserved:%s", msg.ID)

	//check if it already exists in database
	exists := storage.Exists(ctx, id)
	if !exists {
		time := time.Now().UTC()
		err := storage.Create(ctx, id, SewagePumpingStationObserved{
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
	} else {
		spo, err := database.Get[SewagePumpingStationObserved](ctx, storage, id)
		if err != nil {
			return err
		}
		// om sparat state inte är samma som inkommande state
		if spo.State != msg.Stopwatch.State {
			//ingen aktiv alert sen tidigare, det som kommer in är true, gör en ny alert
			if spo.ActiveAlert == "" {
				alert := Alert{
					ID:          "generateAnAlertID",
					AlertSource: spo.ID,
					State:       msg.Stopwatch.State,
					StartTime:   &msg.Stopwatch.StartTime,
				}
				storage.Create(ctx, id, alert)

				spo.ActiveAlert = alert.ID
				spo.LastObserved = alert.StartTime

				storage.Update(ctx, id, spo)
				//TODO: send created alert on queue

			} else { // Om alertId finns, dvs alert finns, stänga alerten
				alert, err := database.Get[Alert](ctx, storage, id)
				if err != nil {
					return err
				}
				alert.State = msg.Stopwatch.State
				alert.EndTime = msg.Stopwatch.StopTime

				storage.Update(ctx, alert.ID, alert)

				spo.PreviousAlerts = append(spo.PreviousAlerts, alert.ID)
				spo.ActiveAlert = ""

				storage.Update(ctx, id, spo)
			}
		} else {
			if spo.State == msg.Stopwatch.State {
				if spo.ActiveAlert != "" {
					alert, err := database.Get[Alert](ctx, storage, id)
					if err != nil {
						return err
					}

					alert.LastObserved = &msg.Timestamp
					storage.Update(ctx, id, alert)

				}
				spo.LastObserved = &msg.Timestamp
				storage.Update(ctx, id, spo)
			}
		}
		//post message to cip-functions.updated
	}

	return nil
}

type Alert struct {
	ID           string     `json:"id"`
	AlertSource  string     `json:"alertSource"`
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
