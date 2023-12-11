package combinedsewageoverflow

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/functions/options"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/cip-functions/pkg/messaging/topics"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const FunctionName string = "combinedsewageoverflow"

type SewageOverflow struct {
	ID       string `json:"id"`
	State    bool   `json:"state"`
	Location Point  `json:"location"`
	Tenant   string `json:"tenant"`
}

func New() SewageOverflow {
	return SewageOverflow{}
}

type SewageOverflowObserved struct {
	ID        string         `json:"id"`
	Count     int32          `json:"count"`
	Duration  *time.Duration `json:"duration,omitempty"`
	EndTime   *time.Time     `json:"endTime,omitempty"`
	State     bool           `json:"state"`
	StartTime *time.Time     `json:"startTime"`
	Timestamp time.Time      `json:"timestamp"`
}

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func (s *SewageOverflow) Handle(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error {
	var err error

	log := logging.GetFromContext(ctx)

	if msg.Type != "stopwatch" && msg.SubType != "overflow" {
		log.Info("invalid function type", "id", msg.ID, "type", msg.Type, "sub_type", msg.SubType)
		return nil
	}

	current, err := database.GetOrDefault[SewageOverflow](ctx, storage, msg.ID, SewageOverflow{
		ID:    msg.ID,
		State: false,
		Location: Point{
			Lat: msg.Location.Latitude,
			Lon: msg.Location.Longitude,
		},
		Tenant: msg.Tenant,
	})
	if err != nil {
		return err
	}

	if current.State != msg.Stopwatch.State {
		current.State = msg.Stopwatch.State

		id := fmt.Sprintf("sewageoverflowobserved:%s:%d", msg.ID, msg.Stopwatch.StartTime.Unix())
		observation, err := database.GetOrDefault[SewageOverflowObserved](ctx, storage, id, SewageOverflowObserved{})
		if err != nil {
			return err
		}

		if msg.Stopwatch.State {
			observation = SewageOverflowObserved{
				ID:        id,
				Count:     msg.Stopwatch.Count,
				Duration:  msg.Stopwatch.Duration,
				State:     msg.Stopwatch.State,
				StartTime: &msg.Stopwatch.StartTime,
				Timestamp: time.Now().UTC(),
			}

			err = storage.Create(ctx, observation.ID, observation)
			if err != nil {
				return err
			}
		} else {
			if observation.EndTime != nil {
				return fmt.Errorf("observation already ended")
			}

			observation.Duration = msg.Stopwatch.Duration
			observation.EndTime = msg.Stopwatch.StopTime
			observation.State = msg.Stopwatch.State
			observation.Timestamp = time.Now().UTC()

			err = storage.Update(ctx, observation.ID, observation)
			if err != nil {
				return err
			}
		}

		err = msgCtx.PublishOnTopic(ctx, observation)
		if err != nil {
			return err
		}
	}

	return database.CreateOrUpdate[SewageOverflow](ctx, storage, current.ID, current)
}

func (s SewageOverflowObserved) Body() []byte {
	// TODO: fill out this function...
	return []byte{}
}

func (s SewageOverflowObserved) TopicName() string {
	return topics.CipFunctionsUpdated
}

func (s SewageOverflowObserved) ContentType() string {
	return "application/vnd+diwise.sewageoverflowobserved+json"
}
