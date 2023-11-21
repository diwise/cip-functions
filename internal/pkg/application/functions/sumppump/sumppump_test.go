package sumppump
/*
import (
	"context"
	"testing"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

func TestSumpPumpHandleCreatesNewIfIDDoesNotExist(t *testing.T) {
	is, dbMock, msgCtxMock := testSetup(t)

	msg := events.FunctionUpdated{
		ID:   "fnID:003",
		Type: "Stopwatch",
		Stopwatch: struct {
			State     bool      "json:\"state\""
			StartTime time.Time "json:\"startTime\""
			StopTime  time.Time "json:\"stopTime,omitempty\""
		}{
			State:     false,
			StartTime: time.Now().UTC(),
		},
	}

	sp := New(dbMock, msgCtxMock)
	err := sp.Handle(context.Background(), &msg)

	is.NoErr(err)
	is.True(len(dbMock.CreateCalls()) == 1)
}

func TestSumpPumpHandleChecksIfStateUpdatedOnExisting(t *testing.T) {
	is, dbMock, msgCtxMock := testSetup(t)

	msg := events.FunctionUpdated{
		ID:   "fnID:004",
		Type: "Stopwatch",
		Stopwatch: struct {
			State     bool      "json:\"state\""
			StartTime time.Time "json:\"startTime\""
			StopTime  time.Time "json:\"stopTime,omitempty\""
		}{
			State:     false,
			StartTime: time.Now().UTC(),
		},
	}

	//create new entry first time around
	sp := New(dbMock, msgCtxMock)
	err := sp.Handle(context.Background(), &msg)
	is.NoErr(err)

	//update value on state
	msg.Stopwatch.State = true

	//call New and Handle again with new value
	sp2 := New(dbMock, msgCtxMock)
	err = sp2.Handle(context.Background(), &msg)

	is.NoErr(err)
	is.True(len(dbMock.UpdateCalls()) == 1)
}

func testSetup(t *testing.T) (*is.I, *database.StorageMock, *messaging.MsgContextMock) {
	is := is.New(t)

	dbMock := &database.StorageMock{
		ExistsFunc: func(ctx context.Context, id string) bool {
			if id == "SumpPumpObserved:fnID:004" {
				return true
			} else {
				return false
			}
		},
		CreateFunc: func(ctx context.Context, id string, value any) error {
			return nil
		},
		UpdateFunc: func(ctx context.Context, id string, value any) error {
			return nil
		},
		SelectFunc: func(ctx context.Context, id string) (any, error) {
			return SumpPumpObserved{
				ID:      "SumpPumpObserved:fnID:004",
				AlertID: "",
				State:   false,
			}, nil
		},
	}

	msgCtxMock := &messaging.MsgContextMock{}

	return is, dbMock, msgCtxMock
}
*/