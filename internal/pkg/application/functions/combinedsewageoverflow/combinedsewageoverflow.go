package combinedsewageoverflow

import (
	"time"

	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
)

type sewageOverflow struct {
	id_ string
	storage database.Storage
}

func New(id string, s database.Storage) *sewageOverflow {
	return &sewageOverflow{
		id_: id,
		storage: s,
	}
}

type sewageOverflowObserved struct {
	ID             string
	Count          int
	CumulativeTime *time.Time
	Description    string
	EndTime        *time.Time
	Location       location
	State          bool
	StartTime      time.Time
	Timestamp      time.Time
}

type location struct {
	lat float64
	lon float64
}

func (s *sewageOverflow) ID() string {
	return s.id_
}

func (s *sewageOverflow) Handle() {
	
}
