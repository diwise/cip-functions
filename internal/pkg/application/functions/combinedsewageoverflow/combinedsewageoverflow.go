package combinedsewageoverflow

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

type SewageOverflow struct {
	ID_ string `json:"id"`

	storage    database.Storage     `json:"-"`
	msgContext messaging.MsgContext `json:"-"`
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

func (s *SewageOverflow) Function() string {
	panic("unimplemented")
}

func (s *SewageOverflow) ID() string {
	return s.ID_
}

func (s *SewageOverflow) Type() string {
	panic("unimplemented")
}

func New(s database.Storage, m messaging.MsgContext) functions.Function {
	return &SewageOverflow{
		storage:    s,
		msgContext: m,
	}
}

func (s *SewageOverflow) Handle(ctx context.Context, msg *events.FunctionUpdated, m messaging.MsgContext) error {
	id := fmt.Sprintf("SewageOverflowObserved:%s", msg.ID()) // TODO: better ID generation

	exists := s.storage.Exists(ctx, id)
	if !exists {
		s.storage.Create(ctx, id, SewageOverflowObserved{
			ID: id,
			// TODO add more fields
		})
	}

	return nil
}

type SewageOverflowObservedStarted struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

func (s SewageOverflowObservedStarted) TopicName() string {
	return "cip-functions.updated"
}

func (s SewageOverflowObservedStarted) ContentType() string {
	return "application/vnd+diwise.sewageoverflowobservedstarted+json"
}
