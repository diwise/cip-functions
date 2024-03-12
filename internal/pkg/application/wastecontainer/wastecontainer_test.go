package wastecontainer

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
	"github.com/diwise/senml"
	"github.com/matryer/is"
)

type messageAccepted struct {
	Pack      senml.Pack `json:"pack"`
	Timestamp time.Time  `json:"timestamp"`
}

func (i messageAccepted) Body() []byte {
	b, _ := json.Marshal(i)
	return b
}
func (i messageAccepted) ContentType() string {
	return "application/vnd.oma.lwm2m.ext.3303"
}
func (i messageAccepted) TopicName() string {
	return "message.accepted"
}

type functionUpdated struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	SubType string `json:"subtype"`
	Level   level  `json:"level,omitempty"`
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

func TestTemperatureHandler(t *testing.T) {
	memStore := make(map[string]any)
	is, msgCtx, tc, s, ctx, log := testSetup(t, memStore)

	var pack senml.Pack
	is.NoErr(json.Unmarshal([]byte(temperature225), &pack))
	ts := time.Unix(1710151647, 0)

	itm := messageAccepted{
		Pack:      pack,
		Timestamp: ts,
	}

	tc.FindRelatedThingsFunc = func(ctx context.Context, thingID string) ([]things.Thing, error) {
		return []things.Thing{
			{
				Id:   "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
				Type: "WasteContainer",
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

	newTemperatureMessageHandler(msgCtx, tc, s)(ctx, itm, log)

	is.Equal(22.5, memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(WasteContainer).Temperature)
}

func TestTemperatureHandlerAnotherPack(t *testing.T) {
	memStore := make(map[string]any)
	is, msgCtx, tc, s, ctx, log := testSetup(t, memStore)

	var pack senml.Pack
	is.NoErr(json.Unmarshal([]byte(temperature22), &pack))
	ts := time.Unix(1710256247, 0)

	itm := messageAccepted{
		Pack:      pack,
		Timestamp: ts,
	}

	tc.FindRelatedThingsFunc = func(ctx context.Context, thingID string) ([]things.Thing, error) {
		return []things.Thing{
			{
				Id:   "72fb1b1c-d574-4946-befe-0ad1ba57bcf5",
				Type: "WasteContainer",
			},
		}, nil
	}

	tc.FindByIDFunc = func(ctx context.Context, thingID string) (things.Thing, error) {
		tenant := "tenant"
		return things.Thing{
			Id:     "72fb1b1c-d574-4946-befe-0ad1ba57bcf5",
			Type:   "WasteContainer",
			Tenant: &tenant,
		}, nil
	}

	msgCtx.PublishOnTopicFunc = func(ctx context.Context, message messaging.TopicMessage) error {
		return nil
	}

	newTemperatureMessageHandler(msgCtx, tc, s)(ctx, itm, log)

	is.Equal(22.0, memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf5"].(WasteContainer).Temperature)
}

func TestLevelHandler(t *testing.T) {
	memStore := make(map[string]any)
	is, msgCtx, tc, s, ctx, log := testSetup(t, memStore)

	var percent float64 = 60
	itm := functionUpdated{
		ID:      "25e185f6-bdba-4c68-b6e8-23ae2bb10254",
		Type:    "level",
		SubType: "overflow",
		Level: level{
			Percent: &percent,
		},
	}

	tc.FindRelatedThingsFunc = func(ctx context.Context, thingID string) ([]things.Thing, error) {
		return []things.Thing{
			{
				Id:   "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
				Type: "WasteContainer",
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

	newLevelMessageHandler(msgCtx, tc, s)(ctx, itm, log)

	is.Equal(60.0, memStore["72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(WasteContainer).Level)
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

	return is, msgCtx, tc, s, context.Background(), log
}

const temperature225 string = `[{"bn":"25e185f6-bdba-4c68-b6e8-23ae2bb10254/3303/","bt":1710151647,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},{"n":"5700","u":"Cel","v":22.5}]`

const temperature22 string = `[{"bn":"f4ee732e-c610-43a5-9bcf-3e4043fe3560/3303/","bt":1710256247,"n":"0","vs":"urn:oma:lwm2m:ext:3303"},{"n":"5700","u":"Cel","v":22},{"u":"lat","v":0},{"u":"lon","v":0},{"n":"env","vs":"indoors"},{"n":"tenant","vs":"default"}]`