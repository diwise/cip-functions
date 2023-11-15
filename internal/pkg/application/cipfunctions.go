package application

import (
	"context"
	"fmt"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/application/messageprocessor"
	"github.com/diwise/cip-functions/pkg/messaging/events"
)

type App interface {
	FunctionUpdated(ctx context.Context, msg events.FunctionUpdated) error
}

type app struct {
	msgproc_   messageprocessor.MessageProcessor
	functions_ functions.Registry
}

func New(msgproc messageprocessor.MessageProcessor, functionRegistry functions.Registry) App {
	return &app{
		msgproc_:   msgproc,
		functions_: functionRegistry,
	}
}

func (a *app) FunctionUpdated(ctx context.Context, msg events.FunctionUpdated) error {
	err := a.msgproc_.ProcessFunctionUpdated(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to process message: %w", err)
	}

	// should return something other than error. Message?
	return nil
}
