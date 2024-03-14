package combinedsewageoverflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

func TestNewStopwatchMessage(t *testing.T) {
	memStore := make(map[string]any)
	is, msgCtx, tc, s, ctx, log := testSetup(t, memStore)

	startTime := time.Now()

	itm := functionUpdated{
		ID:      "25e185f6-bdba-4c68-b6e8-23ae2bb10254",
		Type:    "Stopwatch",
		SubType: "overflow",
		Stopwatch: stopwatch{
			StartTime: startTime,
			State:     true,
		},
	}

	newStopwatchMessageHandler(msgCtx, tc, s)(ctx, itm, log)

	is.True(memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).Overflows[0].State)
}

func TestStateChange(t *testing.T) {
	memStore := make(map[string]any)
	is, msgCtx, tc, s, ctx, log := testSetup(t, memStore)

	stopTime := time.Now()
	startTime := stopTime.Add(-1 * time.Hour)

	itm := functionUpdated{
		ID:      "25e185f6-bdba-4c68-b6e8-23ae2bb10254",
		Type:    "Stopwatch",
		SubType: "overflow",
		Stopwatch: stopwatch{
			StartTime: startTime,
			State:     true,
		},
	}

	newStopwatchMessageHandler(msgCtx, tc, s)(ctx, itm, log)
	is.True(memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).Overflows[0].State)

	itm.Stopwatch.State = false
	itm.Stopwatch.StopTime = &stopTime

	newStopwatchMessageHandler(msgCtx, tc, s)(ctx, itm, log)
	is.True(!memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).Overflows[0].State)
	is.Equal(memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).CumulativeTime, 1*time.Hour)
}

func TestMultipleOverflows(t *testing.T) {
	memStore := make(map[string]any)
	is, msgCtx, tc, s, ctx, log := testSetup(t, memStore)

	stopTime := time.Now()
	startTime := stopTime.Add(-1 * time.Hour)

	itm := functionUpdated{
		ID:      "25e185f6-bdba-4c68-b6e8-23ae2bb10254",
		Type:    "Stopwatch",
		SubType: "overflow",
		Stopwatch: stopwatch{
			StartTime: startTime,
			State:     true,
		},
	}

	newStopwatchMessageHandler(msgCtx, tc, s)(ctx, itm, log)
	is.True(memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).Overflows[0].State)

	itm.Stopwatch.State = false
	itm.Stopwatch.StopTime = &stopTime

	newStopwatchMessageHandler(msgCtx, tc, s)(ctx, itm, log)
	is.True(!memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).Overflows[0].State)
	is.Equal(memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).CumulativeTime, 1*time.Hour)

	itm.Stopwatch.StartTime = stopTime.Add(1 * time.Hour)
	itm.Stopwatch.State = true
	itm.Stopwatch.StopTime = nil

	newStopwatchMessageHandler(msgCtx, tc, s)(ctx, itm, log)
	is.Equal(2, len(memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).Overflows))

	newStopTime := stopTime.Add(2 * time.Hour)
	itm.Stopwatch.StartTime = stopTime.Add(1 * time.Hour)
	itm.Stopwatch.State = false
	itm.Stopwatch.StopTime = &newStopTime

	newStopwatchMessageHandler(msgCtx, tc, s)(ctx, itm, log)
	is.Equal(2, len(memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).Overflows))
	is.Equal(memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(CombinedSewageOverflow).CumulativeTime, 2*time.Hour)
}

func (f functionUpdated) Body() []byte {
	b, _ := json.Marshal(f)
	return b
}
func (f functionUpdated) ContentType() string {
	return "application/vnd.diwise.stopwatch.overflow+json"
}
func (f functionUpdated) TopicName() string {
	return "function.updated"
}

func testSetup(t *testing.T, store map[string]any) (*is.I, *messaging.MsgContextMock, *things.ClientMock, *storage.StorageMock, context.Context, *slog.Logger) {
	is := is.New(t)
	msgCtx := &messaging.MsgContextMock{}
	tc := &things.ClientMock{}
	s := &storage.StorageMock{}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	s.ReadFunc = func(ctx context.Context, id string) (any, error) {
		if a, ok := store[id]; ok {
			return a, nil
		}

		return nil, fmt.Errorf("not found")
	}

	s.ExistsFunc = func(ctx context.Context, id string) bool {
		_, ok := store[id]
		return ok
	}

	s.CreateFunc = func(ctx context.Context, id string, value any) error {
		store[id] = value
		return nil
	}

	s.UpdateFunc = func(ctx context.Context, id string, value any) error {
		store[id] = value
		return nil
	}

	tc.FindRelatedThingsFunc = func(ctx context.Context, thingID string) ([]things.Thing, error) {
		return []things.Thing{
			{
				Id:   "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
				Type: "CombinedSewageOverflow",
			},
		}, nil
	}

	tc.FindByIDFunc = func(ctx context.Context, thingID string) (things.Thing, error) {
		tenant := "tenant"
		return things.Thing{
			Id:     "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
			Type:   "CombinedSewageOverflow",
			Tenant: &tenant,
		}, nil
	}

	msgCtx.PublishOnTopicFunc = func(ctx context.Context, message messaging.TopicMessage) error {
		return nil
	}

	return is, msgCtx, tc, s, context.Background(), log
}
