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

	app, _ := New(msgCtx, tc, s)

	handleFunctionUpdatedMessage(ctx, app, itm, func(id, tenant string) *wastecontainer.WasteContainer {
		return &wastecontainer.WasteContainer{
			ID:     id,
			Type:   "WasteContainer",
			Tenant: tenant,
		}
	}, log)

	is.Equal(60.0, *memStore["WasteContainer:72fb1b1c-d574-4946-befe-0ad1ba57bcf4"].(*wastecontainer.WasteContainer).Percent)
}

func TestCombinedSewageOverflowIntegrationTest(t *testing.T) {
	is, msgCtx, tc, s, ctx, ok := setupIntegrationTest(t)
	if !ok {
		t.Skip()
	}

	app, _ := New(msgCtx, tc, s)

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

	_, err := processIncomingTopicMessage(ctx, app, "25e185f6-bdba-4c68-b6e8-23ae2bb10254", "Stopwatch", itm, combinedsewageoverflow.CombinedSewageOverflowFactory)
	is.NoErr(err)
}

func newTestMessage(contentType string, body string) testMessage {
	var b any
	json.Unmarshal([]byte(body), &b)

	return testMessage{
		body:        b,
		contentType: contentType,
	}
}

type testMessage struct {
	contentType string
	body        any
}

func (m testMessage) ContentType() string {
	return m.contentType
}
func (m testMessage) Body() []byte {
	b, _ := json.Marshal(m.body)
	return b
}
func (m testMessage) TopicName() string {
	return "function.updated"
}

func TestCombinedSewageOverflow(t *testing.T) {
	is, msgCtx, tc, s, ctx, ok := setupIntegrationTest(t)
	if !ok {
		t.Skip()
	}
	app, _ := New(msgCtx, tc, s)
	for _, m := range function_updated_stopwatch {
		itm := newTestMessage("application/vnd.diwise.stopwatch.overflow+json", m)
		_, err := processIncomingTopicMessage(ctx, app, "xyz123", "stopwatch", itm, combinedsewageoverflow.CombinedSewageOverflowFactory)
		is.NoErr(err)
	}
}

func setupIntegrationTest(t *testing.T) (*is.I, *messaging.MsgContextMock, *things.ClientMock, storage.Storage, context.Context, bool) {
	is := is.New(t)
	ctx := context.Background()
	msgCtx := &messaging.MsgContextMock{}
	tc := &things.ClientMock{}

	tc.FindRelatedThingsFunc = func(ctx context.Context, id, thingType string) ([]things.Thing, error) {
		tenant := "tenant"
		return []things.Thing{
			{
				ID:     "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
				Type:   "CombinedSewageOverflow",
				Tenant: tenant,
			},
		}, nil
	}

	tc.FindByIDFunc = func(ctx context.Context, id, thingType string) (things.Thing, error) {
		tenant := "tenant"
		return things.Thing{
			ID:     "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
			Type:   "CombinedSewageOverflow",
			Tenant: tenant,
		}, nil
	}

	msgCtx.PublishOnTopicFunc = func(ctx context.Context, message messaging.TopicMessage) error {
		fmt.Printf("%s\n", string(message.Body()))
		return nil
	}

	msgCtx.RegisterTopicMessageHandlerFunc = func(routingKey string, handler messaging.TopicMessageHandler) error {
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

	tc.FindRelatedThingsFunc = func(ctx context.Context, id, thingType string) ([]things.Thing, error) {
		tenant := "tenant"
		return []things.Thing{
			{
				ID:     "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
				Type:   "WasteContainer",
				Tenant: tenant,
			},
		}, nil
	}

	tc.FindByIDFunc = func(ctx context.Context, id, thingType string) (things.Thing, error) {
		tenant := "tenant"
		return things.Thing{
			ID:     "72fb1b1c-d574-4946-befe-0ad1ba57bcf4",
			Type:   "WasteContainer",
			Tenant: tenant,
		}, nil
	}

	msgCtx.RegisterTopicMessageHandlerFunc = func(routingKey string, handler messaging.TopicMessageHandler) error {
		return nil
	}

	msgCtx.PublishOnTopicFunc = func(ctx context.Context, message messaging.TopicMessage) error {
		return nil
	}

	return is, msgCtx, tc, s, context.Background(), log
}

var function_updated_stopwatch = [5]string{
	`{"id":"xyz123","name":"Förrådet BPN","type":"stopwatch","subtype":"overflow","deviceID":"abc123","tenant":"default","onupdate":true,"timestamp":"2024-08-08T11:21:25.213602292Z","stopwatch":{"startTime":"0001-01-01T00:00:00Z","state":false,"count":0,"cumulativeTime":0}}`,
	`{"id":"xyz123","name":"Förrådet BPN","type":"stopwatch","subtype":"overflow","deviceID":"abc123","tenant":"default","onupdate":true,"timestamp":"2024-08-08T11:21:25.213721769Z","stopwatch":{"startTime":"0001-01-01T00:00:00Z","state":false,"count":0,"cumulativeTime":0}}`,
	`{"id":"xyz123","name":"Förrådet BPN","type":"stopwatch","subtype":"overflow","deviceID":"abc123","tenant":"default","onupdate":true,"timestamp":"2024-08-08T11:21:25.213721769Z","stopwatch":{"startTime":"2024-08-08T09:21:25Z","state":true,"count":1,"cumulativeTime":0}}`,
	`{"id":"xyz123","name":"Förrådet BPN","type":"stopwatch","subtype":"overflow","deviceID":"abc123","tenant":"default","onupdate":true,"timestamp":"2024-08-08T11:21:25.213721769Z","stopwatch":{"startTime":"2024-08-08T09:21:25Z","duration":3600000000000,"state":true,"count":2,"cumulativeTime":0}}`,
	`{"id":"xyz123","name":"Förrådet BPN","type":"stopwatch","subtype":"overflow","deviceID":"abc123","tenant":"default","onupdate":true,"timestamp":"2024-08-08T11:21:25.213721769Z","stopwatch":{"startTime":"2024-08-08T09:21:25Z","stopTime":"2024-08-08T11:21:25Z","duration":7200000000000,"state":false,"count":3,"cumulativeTime":7200000000000}}`,
}
