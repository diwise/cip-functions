// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package things

import (
	"context"
	"sync"
)

// Ensure, that ClientMock does implement Client.
// If this is not the case, regenerate this file with moq.
var _ Client = &ClientMock{}

// ClientMock is a mock implementation of Client.
//
//	func TestSomethingThatUsesClient(t *testing.T) {
//
//		// make and configure a mocked Client
//		mockedClient := &ClientMock{
//			FindRelatedThingsFunc: func(ctx context.Context, thingID string) ([]Thing, error) {
//				panic("mock out the FindRelatedThings method")
//			},
//		}
//
//		// use mockedClient in code that requires Client
//		// and then make assertions.
//
//	}
type ClientMock struct {
	// FindRelatedThingsFunc mocks the FindRelatedThings method.
	FindRelatedThingsFunc func(ctx context.Context, thingID string) ([]Thing, error)

	// calls tracks calls to the methods.
	calls struct {
		// FindRelatedThings holds details about calls to the FindRelatedThings method.
		FindRelatedThings []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ThingID is the thingID argument value.
			ThingID string
		}
	}
	lockFindRelatedThings sync.RWMutex
}

// FindRelatedThings calls FindRelatedThingsFunc.
func (mock *ClientMock) FindRelatedThings(ctx context.Context, thingID string) ([]Thing, error) {
	if mock.FindRelatedThingsFunc == nil {
		panic("ClientMock.FindRelatedThingsFunc: method is nil but Client.FindRelatedThings was just called")
	}
	callInfo := struct {
		Ctx     context.Context
		ThingID string
	}{
		Ctx:     ctx,
		ThingID: thingID,
	}
	mock.lockFindRelatedThings.Lock()
	mock.calls.FindRelatedThings = append(mock.calls.FindRelatedThings, callInfo)
	mock.lockFindRelatedThings.Unlock()
	return mock.FindRelatedThingsFunc(ctx, thingID)
}

// FindRelatedThingsCalls gets all the calls that were made to FindRelatedThings.
// Check the length with:
//
//	len(mockedClient.FindRelatedThingsCalls())
func (mock *ClientMock) FindRelatedThingsCalls() []struct {
	Ctx     context.Context
	ThingID string
} {
	var calls []struct {
		Ctx     context.Context
		ThingID string
	}
	mock.lockFindRelatedThings.RLock()
	calls = mock.calls.FindRelatedThings
	mock.lockFindRelatedThings.RUnlock()
	return calls
}