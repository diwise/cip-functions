package application

import (
	"context"

	"github.com/diwise/cip-functions/internal/pkg/application/registry"
	"github.com/diwise/cip-functions/pkg/messaging/events"
)
//go:generate moq -rm -out app_mock.go . App
type App interface {
	FunctionUpdated(ctx context.Context, msg events.FunctionUpdated) error
}

type app struct {
	fnRegistry registry.Registry
}

func New(functionRegistry registry.Registry) App {
	return &app{
		fnRegistry: functionRegistry,
	}
}

func (a *app) FunctionUpdated(ctx context.Context, msg events.FunctionUpdated) error {

	//TODO: get function from registry and call Handle on it

	a.fnRegistry.Find(ctx, registry.FindByID(msg.ID))

	return nil
}
