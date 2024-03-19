package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/combinedsewageoverflow"
	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/application/wastecontainer"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage/database"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

type functionUpdated struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	SubType   string    `json:"subtype"`
	Level     level     `json:"level,omitempty"`
	Stopwatch stopwatch `json:"stopwatch,omitempty"`
}

type level struct {
	Current float64  `json:"current"`
	Percent *float64 `json:"percent,omitempty"`
}

type stopwatch struct {
	StartTime time.Time      `json:"startTime"`
	StopTime  *time.Time     `json:"stopTime,omitempty"`
	Duration  *time.Duration `json:"duration,omitempty"`

	State bool  `json:"state"`
	Count int32 `json:"count"`

	CumulativeTime time.Duration `json:"cumulativeTime"`
}

func (f functionUpdated) Body() []byte {
	b, _ := json.Marshal(f)
	return b
}
func (f functionUpdated) ContentType() string {
	return "application/vnd.diwise.level.overflow+json"
}
func (f functionUpdated) TopicName() string {
	return "function.updated"
}

func TestGenericHandler(t *testing.T) {
	memStore := make(map[string]any)
	is, msgCtx, tc, s, ctx, log := setup(t, memStore)

	var percent float64 = 60
	itm := functionUpdated{
		ID:      "25e185f6-bdba-4c68-b6e8-23ae2bb10254",
		Type:    "level",
		SubType: "overflow",
		Level: level{
			Percent: &percent,
		},
	}

	app := New(msgCtx, tc, s)

	newFunctionUpdatedHandler(app, func(id, tenant string) *wastecontainer.WasteContainer {
		return &wastecontainer.WasteContainer{
			ID:     id,
			Type:   "WasteContainer",
			Tenant: tenant,
		}
	})(ctx, itm, log)

	is.Equal(60.0, memStore["WasteContainer:72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(*wastecontainer.WasteContainer).Level)
}

func TestCombinedSewageOverflowIntegrationTest(t *testing.T) {
	is, msgCtx, tc, s, ctx, ok := setupIntegrationTest(t)
	if !ok {
		t.Skip()
	}

	app := New(msgCtx, tc, s)

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

	_, err := process(ctx, app, "25e185f6-bdba-4c68-b6e8-23ae2bb10254", itm, combinedsewageoverflow.CombinedSewageOverflowFactory)
	is.NoErr(err)
}

func setupIntegrationTest(t *testing.T) (*is.I, *messaging.MsgContextMock, *things.ClientMock, storage.Storage, context.Context, bool) {
	is := is.New(t)
	ctx := context.Background()
	msgCtx := &messaging.MsgContextMock{}
	tc := &things.ClientMock{}

	tc.FindRelatedThingsFunc = func(ctx context.Context, thingID string) ([]things.Thing, error) {
		tenant := "tenant"
		return []things.Thing{
			{
				Id:     "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
				Type:   "CombinedSewageOverflow",
				Tenant: &tenant,
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

	cfg := database.NewConfig("localhost", "postgres", "postgres", "5432", "postgres", "disable")
	db, err := database.Connect(ctx, cfg)
	if err != nil {
		return is, nil, nil, nil, nil, false
	}

	err = db.Initialize(ctx)
	if err != nil {
		return is, nil, nil, nil, nil, false
	}

	return is, msgCtx, tc, db, ctx, true
}

func setup(t *testing.T, store map[string]any) (*is.I, *messaging.MsgContextMock, *things.ClientMock, *storage.StorageMock, context.Context, *slog.Logger) {
	is := is.New(t)
	msgCtx := &messaging.MsgContextMock{}
	tc := &things.ClientMock{}
	s := &storage.StorageMock{}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	s.ReadFunc = func(ctx context.Context, id, typeName string) (any, error) {
		fullID := fmt.Sprintf("%s:%s", typeName, id)
		if a, ok := store[fullID]; ok {
			return a, nil
		}
		return nil, fmt.Errorf("not found")
	}

	s.ExistsFunc = func(ctx context.Context, id, typeName string) bool {
		fullID := fmt.Sprintf("%s:%s", typeName, id)
		_, ok := store[fullID]
		return ok
	}

	s.CreateFunc = func(ctx context.Context, id, tn string, value any) error {
		fullID := fmt.Sprintf("%s:%s", tn, id)
		store[fullID] = value
		return nil
	}

	tc.FindRelatedThingsFunc = func(ctx context.Context, thingID string) ([]things.Thing, error) {
		tenant := "tenant"
		return []things.Thing{
			{
				Id:     "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
				Type:   "WasteContainer",
				Tenant: &tenant,
			},
		}, nil
	}

	tc.FindByIDFunc = func(ctx context.Context, thingID string) (things.Thing, error) {
		tenant := "tenant"
		return things.Thing{
			Id:     "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
			Type:   "WasteContainer",
			Tenant: &tenant,
		}, nil
	}

	msgCtx.PublishOnTopicFunc = func(ctx context.Context, message messaging.TopicMessage) error {
		return nil
	}

	return is, msgCtx, tc, s, context.Background(), log
}
