package application

import (
	"context"
	"fmt"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/application/messageprocessor"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type App interface {
	MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error
	MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
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

func (a *app) MessageAccepted(ctx context.Context, evt events.MessageAccepted, msgctx messaging.MsgContext) error {
	matchingFunctions, _ := a.functions_.Find(ctx, functions.MatchFunction(evt.ID)) //adding MatchFunction soon.

	logger := logging.GetFromContext(ctx)
	matchingCount := len(matchingFunctions)

	if matchingCount > 0 {
		logger.Debug("found matching functions", "count", matchingCount)
	} else {
		logger.Debug("no matching functions found")
	}

	for _, f := range matchingFunctions {
		if err := f.Handle(ctx, &evt, msgctx); err != nil {
			return err
		}
	}

	return nil
}

func (a *app) MessageReceived(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	messageAccepted, err := a.msgproc_.ProcessMessage(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to process message: %w", err)
	}

	return messageAccepted, nil
}
