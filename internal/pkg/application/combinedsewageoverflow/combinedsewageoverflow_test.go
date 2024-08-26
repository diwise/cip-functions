package combinedsewageoverflow

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/matryer/is"
)

func (f functionUpdated) Body() []byte {
	b, _ := json.Marshal(f)
	return b
}
func (f functionUpdated) ContentType() string {
	return ""
}
func (f functionUpdated) TopicName() string {
	return ""
}

func TestStartStop(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	tc := &things.ClientMock{
		FindByIDFunc: func(ctx context.Context, id, thingType string) (things.Thing, error) {
			return things.Thing{
				ID:   "cso:1",
				Type: "CombinedSewageOverflow",
			}, nil
		},
	}

	startTime := time.Date(2024, 4, 17, 15, 0, 0, 0, time.UTC)

	stopwatch1 := functionUpdated{
		ID:      "sw:1",
		Type:    "Stopwatch",
		SubType: "",
		Stopwatch: stopwatch{
			StartTime: startTime,
			State:     true,
		},
	}

	cso := CombinedSewageOverflow{
		ID:     "cso:1",
		Type:   "CombinedSewageOverflow",
		Tenant: "default",
	}

	changed, err := cso.Handle(ctx, stopwatch1, tc)
	is.NoErr(err)
	is.True(changed)

	stopTime := time.Date(2024, 4, 17, 15, 15, 0, 0, time.UTC)
	stopwatch1.Stopwatch.StopTime = &stopTime
	stopwatch1.Stopwatch.State = false

	changed, err = cso.Handle(ctx, stopwatch1, tc)
	is.NoErr(err)
	is.True(changed)
}

func TestMultipleStopwatches(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	tc := &things.ClientMock{
		FindByIDFunc: func(ctx context.Context, id, thingType string) (things.Thing, error) {
			return things.Thing{
				ID:   "cso:1",
				Type: "CombinedSewageOverflow",
			}, nil
		},
	}

	startTime := time.Date(2024, 4, 17, 15, 0, 0, 0, time.UTC)

	stopwatch1 := functionUpdated{
		ID:      "sw:1",
		Type:    "Stopwatch",
		SubType: "",
		Stopwatch: stopwatch{
			StartTime: startTime,
			State:     true,
		},
	}

	stopwatch2 := functionUpdated{
		ID:      "sw:2",
		Type:    "Stopwatch",
		SubType: "",
		Stopwatch: stopwatch{
			StartTime: startTime.Add(5 * time.Minute),
			State:     true,
		},
	}

	cso := CombinedSewageOverflow{
		ID:     "cso:1",
		Type:   "CombinedSewageOverflow",
		Tenant: "default",
	}

	changed, err := cso.Handle(ctx, stopwatch1, tc)
	is.NoErr(err)
	is.True(changed)

	changed, err = cso.Handle(ctx, stopwatch2, tc)
	is.NoErr(err)
	is.True(changed)

	stopTime := time.Date(2024, 4, 17, 15, 15, 0, 0, time.UTC)
	stopwatch1.Stopwatch.StopTime = &stopTime
	stopwatch1.Stopwatch.State = false

	changed, err = cso.Handle(ctx, stopwatch1, tc)
	is.NoErr(err)
	is.True(changed)
	is.True(cso.State)
}
