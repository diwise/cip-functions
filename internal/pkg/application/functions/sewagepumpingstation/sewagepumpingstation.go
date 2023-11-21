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

type SewagePumpingStation interface {
	Handle(ctx context.Context, msg *events.FunctionUpdated, options ...options.Option) error
}

type sp struct {
	storage database.Storage
	msgCtx  messaging.MsgContext
}

func New(storage database.Storage, msgCtx messaging.MsgContext) SewagePumpingStation {
	sp := &sp{
		storage: storage,
		msgCtx:  msgCtx,
	}

	return sp
}

type SewagePumpingStationObserved struct {
	ID          string `json:"id"`
	ActiveAlert struct {
		AlertID string `json:"alertId,omitempty"`
	} `json:"activeAlert,omitempty"`
	PreviousAlerts []string   `json:"previousAlerts,omitempty"`
	State          bool       `json:"state"`
	StartTime      *time.Time `json:"startTime,omitempty"`
	EndTime        *time.Time `json:"endTime,omitempty"`
	LastObserved   *time.Time `json:"lastObserved"`
}

func (sp *sp) Handle(ctx context.Context, msg *events.FunctionUpdated, options ...options.Option) error {
	//generate ID
	id := fmt.Sprintf("SewagePumpingStationObserved:%s", msg.ID)

	//check if it already exists in database
	exists := sp.storage.Exists(ctx, id)
	if !exists {
		time := time.Now().UTC()
		err := sp.storage.Create(ctx, id, SewagePumpingStationObserved{
			ID:           id,
			State:        msg.Stopwatch.State,
			LastObserved: &time,
		})
		if err != nil {
			return err
		}

		if msg.Stopwatch.State {
			alertID := fmt.Sprintf("Alert:alertID:%s", id) //TODO: better ID generation
			err = sp.storage.Create(ctx, fmt.Sprintf("%s:alert:", id), Alert{
				ID:        alertID,
				State:     msg.Stopwatch.State,
				StartTime: &msg.Stopwatch.StartTime,
			})
			if err != nil {
				return err
			}
		}

	} else if exists {
		spo := SewagePumpingStationObserved{}

		spo, err := database.Get[SewagePumpingStationObserved](ctx, sp.storage, id)
		if err != nil {
			return err
		}

		//if it already does and state has not changed, update dateModified (false-false, true-true)

		//if it already exists, and state has now changed create a new pumpbrunn/alert
		// kolla om det finns en aktiv alert om nej skapa en ny alert
		// om aktiv alert finns uppdatera alert och ta bort alertId på pumpbrunn - gäller true to false
		if spo.State != msg.Stopwatch.State {
			if spo.ActiveAlert.AlertID == "" {
				alert := Alert{
					ID:          "generateAnAlertID",
					AlertSource: spo.ID,
					State:       msg.Stopwatch.State,
					StartTime:   &msg.Stopwatch.StartTime,
				}
				sp.storage.Create(ctx, id, alert)

				spo.ActiveAlert.AlertID = alert.ID
				spo.LastObserved = alert.StartTime

				sp.storage.Update(ctx, id, spo)
				//TODO: send created alert on queue

			} else { // Om alertId finns, dvs alert finns, stänga alerten
				alert := Alert{}
				alert, err = database.Get[Alert](ctx, sp.storage, id)
				if err != nil {
					return err
				}
				alert.State = msg.Stopwatch.State
				alert.StartTime = &msg.Stopwatch.StopTime
				spo.PreviousAlerts = append(spo.PreviousAlerts, alert.ID)
				spo.ActiveAlert.AlertID = ""

				//update existing alert with new timestamp and/or endTime
				//spo.LastObserved = msg.LastObserved or something like that.
				sp.storage.Update(ctx, id, spo)
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
