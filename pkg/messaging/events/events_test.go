package events

import (
	"encoding/json"
	"testing"

	"github.com/matryer/is"
)

func TestThatTypeCanBeRetrievedFromMessage(t *testing.T) {
	is := testSetup(t)

	msgReceived := MessageReceived{}

	err := json.Unmarshal([]byte(incomingMsg), &msgReceived)
	is.NoErr(err)

	newMsg := NewMessageAccepted(msgReceived.ID_, msgReceived)

	is.Equal(newMsg.FunctionType(), "Stopwatch")
}

func testSetup(t *testing.T) *is.I {
	is := is.New(t)
	return is
}

const incomingMsg string = `{"id":"functionID","name":"name","type":"Stopwatch","Stopwatch":{"state":0,"timestamp":"2023-06-05T11:26:57Z"}}`
