// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package storage

import (
	"context"
	"sync"
)

// Ensure, that StorageMock does implement Storage.
// If this is not the case, regenerate this file with moq.
var _ Storage = &StorageMock{}

// StorageMock is a mock implementation of Storage.
//
//	func TestSomethingThatUsesStorage(t *testing.T) {
//
//		// make and configure a mocked Storage
//		mockedStorage := &StorageMock{
//			CreateFunc: func(ctx context.Context, id string, typeName string, value any) error {
//				panic("mock out the Create method")
//			},
//			DeleteFunc: func(ctx context.Context, id string, typeName string) error {
//				panic("mock out the Delete method")
//			},
//			ExistsFunc: func(ctx context.Context, id string, typeName string) bool {
//				panic("mock out the Exists method")
//			},
//			ReadFunc: func(ctx context.Context, id string, typeName string) (any, error) {
//				panic("mock out the Read method")
//			},
//			UpdateFunc: func(ctx context.Context, id string, typeName string, value any) error {
//				panic("mock out the Update method")
//			},
//		}
//
//		// use mockedStorage in code that requires Storage
//		// and then make assertions.
//
//	}
type StorageMock struct {
	// CreateFunc mocks the Create method.
	CreateFunc func(ctx context.Context, id string, typeName string, value any) error

	// DeleteFunc mocks the Delete method.
	DeleteFunc func(ctx context.Context, id string, typeName string) error

	// ExistsFunc mocks the Exists method.
	ExistsFunc func(ctx context.Context, id string, typeName string) bool

	// ReadFunc mocks the Read method.
	ReadFunc func(ctx context.Context, id string, typeName string) (any, error)

	// UpdateFunc mocks the Update method.
	UpdateFunc func(ctx context.Context, id string, typeName string, value any) error

	// calls tracks calls to the methods.
	calls struct {
		// Create holds details about calls to the Create method.
		Create []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// TypeName is the typeName argument value.
			TypeName string
			// Value is the value argument value.
			Value any
		}
		// Delete holds details about calls to the Delete method.
		Delete []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// TypeName is the typeName argument value.
			TypeName string
		}
		// Exists holds details about calls to the Exists method.
		Exists []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// TypeName is the typeName argument value.
			TypeName string
		}
		// Read holds details about calls to the Read method.
		Read []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// TypeName is the typeName argument value.
			TypeName string
		}
		// Update holds details about calls to the Update method.
		Update []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// ID is the id argument value.
			ID string
			// TypeName is the typeName argument value.
			TypeName string
			// Value is the value argument value.
			Value any
		}
	}
	lockCreate sync.RWMutex
	lockDelete sync.RWMutex
	lockExists sync.RWMutex
	lockRead   sync.RWMutex
	lockUpdate sync.RWMutex
}

// Create calls CreateFunc.
func (mock *StorageMock) Create(ctx context.Context, id string, typeName string, value any) error {
	if mock.CreateFunc == nil {
		panic("StorageMock.CreateFunc: method is nil but Storage.Create was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		ID       string
		TypeName string
		Value    any
	}{
		Ctx:      ctx,
		ID:       id,
		TypeName: typeName,
		Value:    value,
	}
	mock.lockCreate.Lock()
	mock.calls.Create = append(mock.calls.Create, callInfo)
	mock.lockCreate.Unlock()
	return mock.CreateFunc(ctx, id, typeName, value)
}

// CreateCalls gets all the calls that were made to Create.
// Check the length with:
//
//	len(mockedStorage.CreateCalls())
func (mock *StorageMock) CreateCalls() []struct {
	Ctx      context.Context
	ID       string
	TypeName string
	Value    any
} {
	var calls []struct {
		Ctx      context.Context
		ID       string
		TypeName string
		Value    any
	}
	mock.lockCreate.RLock()
	calls = mock.calls.Create
	mock.lockCreate.RUnlock()
	return calls
}

// Delete calls DeleteFunc.
func (mock *StorageMock) Delete(ctx context.Context, id string, typeName string) error {
	if mock.DeleteFunc == nil {
		panic("StorageMock.DeleteFunc: method is nil but Storage.Delete was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		ID       string
		TypeName string
	}{
		Ctx:      ctx,
		ID:       id,
		TypeName: typeName,
	}
	mock.lockDelete.Lock()
	mock.calls.Delete = append(mock.calls.Delete, callInfo)
	mock.lockDelete.Unlock()
	return mock.DeleteFunc(ctx, id, typeName)
}

// DeleteCalls gets all the calls that were made to Delete.
// Check the length with:
//
//	len(mockedStorage.DeleteCalls())
func (mock *StorageMock) DeleteCalls() []struct {
	Ctx      context.Context
	ID       string
	TypeName string
} {
	var calls []struct {
		Ctx      context.Context
		ID       string
		TypeName string
	}
	mock.lockDelete.RLock()
	calls = mock.calls.Delete
	mock.lockDelete.RUnlock()
	return calls
}

// Exists calls ExistsFunc.
func (mock *StorageMock) Exists(ctx context.Context, id string, typeName string) bool {
	if mock.ExistsFunc == nil {
		panic("StorageMock.ExistsFunc: method is nil but Storage.Exists was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		ID       string
		TypeName string
	}{
		Ctx:      ctx,
		ID:       id,
		TypeName: typeName,
	}
	mock.lockExists.Lock()
	mock.calls.Exists = append(mock.calls.Exists, callInfo)
	mock.lockExists.Unlock()
	return mock.ExistsFunc(ctx, id, typeName)
}

// ExistsCalls gets all the calls that were made to Exists.
// Check the length with:
//
//	len(mockedStorage.ExistsCalls())
func (mock *StorageMock) ExistsCalls() []struct {
	Ctx      context.Context
	ID       string
	TypeName string
} {
	var calls []struct {
		Ctx      context.Context
		ID       string
		TypeName string
	}
	mock.lockExists.RLock()
	calls = mock.calls.Exists
	mock.lockExists.RUnlock()
	return calls
}

// Read calls ReadFunc.
func (mock *StorageMock) Read(ctx context.Context, id string, typeName string) (any, error) {
	if mock.ReadFunc == nil {
		panic("StorageMock.ReadFunc: method is nil but Storage.Read was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		ID       string
		TypeName string
	}{
		Ctx:      ctx,
		ID:       id,
		TypeName: typeName,
	}
	mock.lockRead.Lock()
	mock.calls.Read = append(mock.calls.Read, callInfo)
	mock.lockRead.Unlock()
	return mock.ReadFunc(ctx, id, typeName)
}

// ReadCalls gets all the calls that were made to Read.
// Check the length with:
//
//	len(mockedStorage.ReadCalls())
func (mock *StorageMock) ReadCalls() []struct {
	Ctx      context.Context
	ID       string
	TypeName string
} {
	var calls []struct {
		Ctx      context.Context
		ID       string
		TypeName string
	}
	mock.lockRead.RLock()
	calls = mock.calls.Read
	mock.lockRead.RUnlock()
	return calls
}

// Update calls UpdateFunc.
func (mock *StorageMock) Update(ctx context.Context, id string, typeName string, value any) error {
	if mock.UpdateFunc == nil {
		panic("StorageMock.UpdateFunc: method is nil but Storage.Update was just called")
	}
	callInfo := struct {
		Ctx      context.Context
		ID       string
		TypeName string
		Value    any
	}{
		Ctx:      ctx,
		ID:       id,
		TypeName: typeName,
		Value:    value,
	}
	mock.lockUpdate.Lock()
	mock.calls.Update = append(mock.calls.Update, callInfo)
	mock.lockUpdate.Unlock()
	return mock.UpdateFunc(ctx, id, typeName, value)
}

// UpdateCalls gets all the calls that were made to Update.
// Check the length with:
//
//	len(mockedStorage.UpdateCalls())
func (mock *StorageMock) UpdateCalls() []struct {
	Ctx      context.Context
	ID       string
	TypeName string
	Value    any
} {
	var calls []struct {
		Ctx      context.Context
		ID       string
		TypeName string
		Value    any
	}
	mock.lockUpdate.RLock()
	calls = mock.calls.Update
	mock.lockUpdate.RUnlock()
	return calls
}
