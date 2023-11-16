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

type SewageOverflow struct {
	storage database.Storage     `json:"-"`
	msgCtx  messaging.MsgContext `json:"-"`
}

func New(s database.Storage, msgctx messaging.MsgContext) SewageOverflow {
	return SewageOverflow{
		storage: s,
		msgCtx:  msgctx,
	}
}

type SewageOverflowObserved struct {
	ID          string     `json:"id"`
	Count       int        `json:"count"`
	Duration    *time.Time `json:"duration,omitempty"`
	Description string     `json:"description"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Location    Point      `json:"location"`
	State       bool       `json:"state"`
	StartTime   *time.Time `json:"startTime"`
	Timestamp   time.Time  `json:"timestamp"`
}

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func (s *SewageOverflow) Handle(ctx context.Context, msg *events.FunctionUpdated, options ...options.Option) error {
	id := fmt.Sprintf("SewageOverflowObserved:%s", msg.ID) // TODO: better ID generation

	exists := s.storage.Exists(ctx, id)
	if !exists {
		s.storage.Create(ctx, id, SewageOverflowObserved{
			ID: id,
			// TODO add more fields
		})
	}

	// TODO: publish events to message broker
	return nil
}

type SewageOverflowObservedStarted struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

func (s SewageOverflowObservedStarted) TopicName() string {
	return topics.CipFunctionsUpdated
}

func (s SewageOverflowObservedStarted) ContentType() string {
	return "application/vnd+diwise.sewageoverflowobservedstarted+json"
}
