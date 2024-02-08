package sewagepumpingstation

import (
	"context"
	"testing"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/matryer/is"
)

func TestSewagePumpingStationHandleCreatesNewSewagePumpingStationIfIDDoesNotExist(t *testing.T) {
	is, dbMock, msgCtxMock, msg := testSetup(t, "fnID:003", false)

	sp := New()
	err := sp.Handle(context.Background(), &msg, dbMock, msgCtxMock)

	is.NoErr(err)
	is.Equal(len(dbMock.CreateCalls()), 1) //create should be called once to create a new sewagepumpingstation
	is.Equal(len(msgCtxMock.PublishOnTopicCalls()), 1)
}

func TestSewagePumpingStationHandleCreatesNewAlertIfFirstStateIsTrue(t *testing.T) {
	is, dbMock, msgCtxMock, msg := testSetup(t, "fnID:003", true)

	sp := New()
	err := sp.Handle(context.Background(), &msg, dbMock, msgCtxMock)

	is.NoErr(err)
	is.Equal(len(dbMock.CreateCalls()), 1) //create should be called once to create a new sewagepumpingstation
	is.Equal(len(msgCtxMock.PublishOnTopicCalls()), 1)
}

func TestSewagePumpingStationHandleChecksIfStateHasUpdatedOnExisting(t *testing.T) {
	is, dbMock, msgCtxMock, msg := testSetup(t, "fnID:004", false)

	//create new entry first time around
	sp := New()
	err := sp.Handle(context.Background(), &msg, dbMock, msgCtxMock)
	is.NoErr(err)

	//update value on state
	msg.State.State_ = true

	//call New and Handle again with new value
	sp2 := New()
	err = sp2.Handle(context.Background(), &msg, dbMock, msgCtxMock)

	is.NoErr(err)
	is.Equal(len(dbMock.UpdateCalls()), 2) //update is called twice, first with timestamp from same state, then with altered state for sewagepumpingstation
	is.Equal(len(msgCtxMock.PublishOnTopicCalls()), 2)
}

func TestSewagePumpingStationHandleChecksIfAlertCloses(t *testing.T) {
	is, dbMock, msgCtxMock, msg := testSetup(t, "fnID:004", true)

	//create new entry first time around
	sp := New()
	err := sp.Handle(context.Background(), &msg, dbMock, msgCtxMock)
	is.NoErr(err)

	//update value on state
	msg.State.State_ = false

	//call New and Handle again with new value
	sp2 := New()
	err = sp2.Handle(context.Background(), &msg, dbMock, msgCtxMock)

	is.NoErr(err)
	is.Equal(len(dbMock.UpdateCalls()), 2)
	is.Equal(len(msgCtxMock.PublishOnTopicCalls()), 2)
}

func TestSewagePumpingStationHandleChecksIfStatusIsUnchanged(t *testing.T) {
	is, dbMock, msgCtxMock, msg := testSetup(t, "fnID:004", true)

	//create new entry first time around
	sp := New()
	err := sp.Handle(context.Background(), &msg, dbMock, msgCtxMock)
	is.NoErr(err)

	//call New and Handle again with new value
	sp2 := New()
	err = sp2.Handle(context.Background(), &msg, dbMock, msgCtxMock)

	is.NoErr(err)
	is.Equal(len(dbMock.UpdateCalls()), 2)
	is.Equal(len(msgCtxMock.PublishOnTopicCalls()), 2)
}

func testSetup(t *testing.T, msgID string, state bool) (*is.I, *database.StorageMock, *messaging.MsgContextMock, events.FunctionUpdated) {
	is := is.New(t)

	timestamp := time.Now()

	msg := events.FunctionUpdated{
		ID:   msgID,
		Type: "state",
		State: struct {
			Timestamp string "json:\"timestamp\""
			State_    bool   "json:\"state\""
		}{

			Timestamp: timestamp.Format(time.RFC3339),
			State_:    state,
		},
	}

	dbMock := &database.StorageMock{
		ExistsFunc: func(ctx context.Context, id string) bool {
			if id == "fnID:004" {
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
			return SewagePumpingStation{
				ID:    id,
				State: state,

				ObservedAt: &timestamp,
			}, nil
		},
	}

	msgCtxMock := &messaging.MsgContextMock{
		PublishOnTopicFunc: func(ctx context.Context, message messaging.TopicMessage) error {
			return nil
		},
	}

	return is, dbMock, msgCtxMock, msg
}
