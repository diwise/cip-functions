package events

import (
	"testing"

	"github.com/matryer/is"
)

func TestThatTypeCanBeRetrievedFromMessage(t *testing.T) {
	// create new test when we know how we want to do this
}

func testSetup(t *testing.T) *is.I {
	is := is.New(t)
	return is
}

const incomingMsg string = `{"id":"functionID","name":"name","type":"stopwatch","stopwatch":{"state":0,"timestamp":"2023-06-05T11:26:57Z"}}`
