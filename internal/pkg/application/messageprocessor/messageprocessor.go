package messageprocessor

import (
	"context"
	"fmt"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/pkg/messaging/events"
)

//go:generate moq -rm -out messageprocessor_mock.go . MessageProcessor

type MessageProcessor interface {
	ProcessFunctionUpdated(ctx context.Context, msg events.FunctionUpdated) error
}

type messageProcessor struct {
}

func NewMessageProcessor() MessageProcessor {
	return &messageProcessor{}
}

func (m *messageProcessor) ProcessFunctionUpdated(ctx context.Context, msg events.FunctionUpdated) error {
	if msg.ID() == "" {
		return fmt.Errorf("message contains no FunctionID")
	}

	//find function from functionID, return and create message? Handle? Handle should probably already have happened at this point.
	_, err := functions.FindCIPFunctionFromFunctionID(ctx, msg.ID())
	if err != nil {
		return fmt.Errorf("could not find cip-function with functionID %s, %w", msg.ID(), err)
	}

	//this function should return outgoing message, but setting it to error only until we know what that message looks like
	return nil
}
