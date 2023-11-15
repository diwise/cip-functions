package events

import (
	"encoding/json"
	"testing"

	"github.com/matryer/is"
)

func TestThatTypeCanBeRetrievedFromMessage(t *testing.T) {
	is := testSetup(t)

	fnctUpdated := FunctionUpdated{}

	err := json.Unmarshal([]byte(incomingMsg), &fnctUpdated)
	is.NoErr(err)

	newMsg := NewMessageAccepted(fnctUpdated.ID_, fnctUpdated)

	is.Equal(newMsg.Type, "Stopwatch")
}

func testSetup(t *testing.T) *is.I {
	is := is.New(t)
	return is
}

const incomingMsg string = `{"id":"functionID","name":"name","type":"Stopwatch","Stopwatch":{"state":0,"timestamp":"2023-06-05T11:26:57Z"}}`
