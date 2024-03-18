package application

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

func TestThatFunctionUpdatedFindsAndUpdatesExistingDigitalInput(t *testing.T) {
	is, ctx, msgCtx, db := testSetup(t)

	fnReg, err := functions.NewRegistry(ctx, bytes.NewBufferString(input))
	is.NoErr(err)

	app := New(db, msgCtx, fnReg)

	timestamp := time.Now()

	msg := events.FunctionUpdated{
		ID:        "digitalinput:002",
		Type:      "digitalinput",
		Timestamp: timestamp,
		Stopwatch: struct {
			Count          int32          "json:\"count\""
			CumulativeTime time.Duration  "json:\"cumulativeTime\""
			Duration       *time.Duration "json:\"duration,omitempty\""
			StartTime      time.Time      "json:\"startTime\""
			State          bool           "json:\"state\""
			StopTime       *time.Time     "json:\"stopTime,omitempty\""
		}{
			State:     false,
			StartTime: time.Time{},
			StopTime:  &time.Time{},
		},
	}

	err = app.FunctionUpdated(ctx, msg)
	is.NoErr(err)
	is.Equal(len(db.CreateCalls()), 0) // this should be zero as id of function updated is set to same string as used in function registry
	is.Equal(len(db.UpdateCalls()), 1)
	is.Equal(len(msgCtx.PublishOnTopicCalls()), 1)
}

func testSetup(t *testing.T) (*is.I, context.Context, *messaging.MsgContextMock, *storage.StorageMock) {
	is := is.New(t)
	ctx := context.Background()
	msgCtx := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error {
			return nil
		},
	}

	db := &storage.StorageMock{
		CreateFunc: func(ctx context.Context, id,tn string, value any) error {
			return nil
		},
		UpdateFunc: func(ctx context.Context, id,tn string, value any) error {
			return nil
		},
		ExistsFunc: func(ctx context.Context, id, typeName string) bool {
			return id == "digitalinput:002"
		},
		ReadFunc: func(ctx context.Context, id, typeName string) (any, error) {
			return nil, nil
		},
	}

	return is, ctx, msgCtx, db
}

const input string = `iot-functionID;cip-function_type;arguments
fnID:001;combinedsewageoverflow;
fnID:002;combinedsewageoverflow;cipID=abc,max=100,min=0
digitalinput:002;sewagepumpingstation;`
