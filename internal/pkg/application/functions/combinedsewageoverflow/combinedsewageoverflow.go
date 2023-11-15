package combinedsewageoverflow

import (
	"context"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

type SewageOverflow struct {
	ID_         string     `json:"id"`
	Count       int        `json:"count"`
	Duration    *time.Time `json:"duration,omitempty"`
	Description string     `json:"description"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Location    Point      `json:"location"`
	State       bool       `json:"state"`
	StartTime   *time.Time `json:"startTime"`
	Timestamp   time.Time  `json:"timestamp"`

	storage    database.Storage     `json:"-"`
	msgContext messaging.MsgContext `json:"-"`
}

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Function implements functions.Function.
func (s *SewageOverflow) Function() string {
	panic("unimplemented")
}

// ID implements functions.Function.
func (s *SewageOverflow) ID() string {
	return s.ID_
}

// Type implements functions.Function.
func (s *SewageOverflow) Type() string {
	panic("unimplemented")
}

func New(s database.Storage, m messaging.MsgContext) functions.Function {
	return &SewageOverflow{
		storage:    s,
		msgContext: m,
	}
}

// Handle implements functions.Function.
func (s *SewageOverflow) Handle(ctx context.Context, msg *events.FunctionUpdated, m messaging.MsgContext) error {
	panic("unimplemented")
}
