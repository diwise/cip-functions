package messageprocessor

import (
	"context"
	"fmt"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/pkg/messaging/events"
)

//go:generate moq -rm -out messageprocessor_mock.go . MessageProcessor

type MessageProcessor interface {
	ProcessFunctionUpdated(ctx context.Context, msg events.FunctionUpdated) (*events.MessageAccepted, error)
}

type messageProcessor struct {
}

func NewMessageProcessor() MessageProcessor {
	return &messageProcessor{}
}

func (m *messageProcessor) ProcessFunctionUpdated(ctx context.Context, msg events.FunctionUpdated) (*events.MessageAccepted, error) {
	if msg.ID() == "" {
		return nil, fmt.Errorf("message contains no FunctionID")
	}

	//find function from functionID
	function, err := functions.FindCIPFunctionFromFunctionID(ctx, msg.ID())
	if err != nil {
		return nil, fmt.Errorf("could not find cip-function with functionID %s, %w", msg.ID(), err)
	}

	return events.NewMessageAccepted(function.ID(), msg), nil
}
