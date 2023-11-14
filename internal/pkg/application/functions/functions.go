package functions

import (
	"context"

	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

type Function interface {
	ID() string
	Function() string
	Type() string

	Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error
}

type location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type fnct struct {
	ID_       string `json:"id"`
	Type_     string `json:"type"`
	Function_ string `json:"function"`
}

func (f *fnct) Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error {
	return nil
}

func (f *fnct) ID() string {
	return f.ID_
}

func (f *fnct) Type() string {
	return f.Type_
}

func (f *fnct) Function() string {
	return f.Function_
}

func FindCIPFunctionFromFunctionID(ctx context.Context, functionID string) (Function, error) {

	//use function ID to find any matching cip-functions,
	//return Function Interfaction that gives ID, Name and Type.
	//Handle might have to contain a check for what type of function has come in, and then call on the relevant function Handler, i.e Stopwatch, Levels, Presence etc.
	//wondering if the handler in each function will determine behaviour of next function

	return nil, nil
}
