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
)

const FunctionName string = "combinedsewageoverflow"

type SewageOverflow struct {
	current SewageOverflowObserved
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

	sufix, ok := options.Exists(opts, "cipID") // TODO: add constant
	if !ok {
		sufix = msg.ID
	}

	id := fmt.Sprintf("SewageOverflowObserved:%s", sufix)

	exists := storage.Exists(ctx, id)
	if !exists {
		s.current = SewageOverflowObserved{
			ID:        id,
			Count:     msg.Stopwatch.Count,
			Duration:  msg.Stopwatch.Duration,
			State:     msg.Stopwatch.State,
			StartTime: &msg.Stopwatch.StartTime,
			Timestamp: time.Now().UTC(),
		}
	} else {
		s.current, err = database.Get[SewageOverflowObserved](ctx, storage, id)
		if err != nil {
			return err
		}
	}

	return msgCtx.PublishOnTopic(ctx, s.current)
}

func (s SewageOverflowObserved) TopicName() string {
	return topics.CipFunctionsUpdated
}

func (s SewageOverflowObserved) ContentType() string {
	return "application/vnd+diwise.SewageOverflowObserved+json"
}
