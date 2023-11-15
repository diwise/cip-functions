// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package messageprocessor

import (
	"context"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"sync"
)

// Ensure, that MessageProcessorMock does implement MessageProcessor.
// If this is not the case, regenerate this file with moq.
var _ MessageProcessor = &MessageProcessorMock{}

// MessageProcessorMock is a mock implementation of MessageProcessor.
//
//	func TestSomethingThatUsesMessageProcessor(t *testing.T) {
//
//		// make and configure a mocked MessageProcessor
//		mockedMessageProcessor := &MessageProcessorMock{
//			ProcessFunctionUpdatedFunc: func(ctx context.Context, msg events.FunctionUpdated) (*events.MessageAccepted, error) {
//				panic("mock out the ProcessFunctionUpdated method")
//			},
//		}
//
//		// use mockedMessageProcessor in code that requires MessageProcessor
//		// and then make assertions.
//
//	}
type MessageProcessorMock struct {
	// ProcessFunctionUpdatedFunc mocks the ProcessFunctionUpdated method.
	ProcessFunctionUpdatedFunc func(ctx context.Context, msg events.FunctionUpdated) (*events.MessageAccepted, error)

	// calls tracks calls to the methods.
	calls struct {
		// ProcessFunctionUpdated holds details about calls to the ProcessFunctionUpdated method.
		ProcessFunctionUpdated []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Msg is the msg argument value.
			Msg events.FunctionUpdated
		}
	}
	lockProcessFunctionUpdated sync.RWMutex
}

// ProcessFunctionUpdated calls ProcessFunctionUpdatedFunc.
func (mock *MessageProcessorMock) ProcessFunctionUpdated(ctx context.Context, msg events.FunctionUpdated) (*events.MessageAccepted, error) {
	if mock.ProcessFunctionUpdatedFunc == nil {
		panic("MessageProcessorMock.ProcessFunctionUpdatedFunc: method is nil but MessageProcessor.ProcessFunctionUpdated was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Msg events.FunctionUpdated
	}{
		Ctx: ctx,
		Msg: msg,
	}
	mock.lockProcessFunctionUpdated.Lock()
	mock.calls.ProcessFunctionUpdated = append(mock.calls.ProcessFunctionUpdated, callInfo)
	mock.lockProcessFunctionUpdated.Unlock()
	return mock.ProcessFunctionUpdatedFunc(ctx, msg)
}

// ProcessFunctionUpdatedCalls gets all the calls that were made to ProcessFunctionUpdated.
// Check the length with:
//
//	len(mockedMessageProcessor.ProcessFunctionUpdatedCalls())
func (mock *MessageProcessorMock) ProcessFunctionUpdatedCalls() []struct {
	Ctx context.Context
	Msg events.FunctionUpdated
} {
	var calls []struct {
		Ctx context.Context
		Msg events.FunctionUpdated
	}
	mock.lockProcessFunctionUpdated.RLock()
	calls = mock.calls.ProcessFunctionUpdated
	mock.lockProcessFunctionUpdated.RUnlock()
	return calls
}
