package functions

import (
	"context"

	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

type Function interface {
	ID() string
	Name() string
	Type() string

	Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error
}

type fnct struct{}

func (f *fnct) Handle(context.Context, *events.MessageAccepted, messaging.MsgContext) error {
	return nil
}

func (f *fnct) ID() string {
	return ""
}

func (f *fnct) Name() string {
	return ""
}

func (f *fnct) Type() string {
	return ""
}

func FindCIPFunctionFromFunctionID(ctx context.Context, functionID string) (Function, error) {
	return nil, nil
}
