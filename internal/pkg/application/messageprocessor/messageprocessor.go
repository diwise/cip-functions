package messageprocessor

import (
	"context"
	"fmt"

	"github.com/diwise/cip-functions/pkg/messaging/events"
)

//go:generate moq -rm -out messageprocessor_mock.go . MessageProcessor

type MessageProcessor interface {
	ProcessMessage(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error)
}

type messageProcessor struct {
}

func NewMessageProcessor() MessageProcessor {
	return &messageProcessor{}
}

func (m *messageProcessor) ProcessMessage(ctx context.Context, msg events.MessageReceived) (*events.MessageAccepted, error) {
	if msg.FunctionID() == "" {
		return nil, fmt.Errorf("message contains no FunctionID")
	}

	//find function from functionID
	/*
			function, err := functions.FindFunctionFromFunctionID(ctx, msg.FunctionID())
			if err != nil {
				return nil, fmt.Errorf("could not find function with functionID %s, %w", msg.FunctionID(), err)
			}


		return events.NewMessageAccepted(device.ID(), msg.Pack.Clone(),
			events.Lat(device.Latitude()),
			events.Lon(device.Longitude()),
			events.Environment(device.Environment()),
			events.Source(device.Source()),
			events.Tenant(device.Tenant())), nil

	*/

	return nil, nil
}
