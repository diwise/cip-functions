package application

import (
	"context"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/application/functions/combinedsewageoverflow"
	"github.com/diwise/cip-functions/internal/pkg/application/functions/sumppump"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

//go:generate moq -rm -out app_mock.go . App
type App interface {
	FunctionUpdated(ctx context.Context, msg events.FunctionUpdated) error
}

type app struct {
	fnRegistry functions.Registry
	storage    database.Storage
	msgCtx     messaging.MsgContext
}

func New(s database.Storage, m messaging.MsgContext, r functions.Registry) App {
	return &app{
		fnRegistry: r,
		storage:    s,
		msgCtx:     m,
	}
}

func (a *app) FunctionUpdated(ctx context.Context, msg events.FunctionUpdated) error {
	log := logging.GetFromContext(ctx)

	//TODO: get function from registry and call Handle on it

	registryItems, err := a.fnRegistry.Find(ctx, functions.FindByFunctionID(msg.ID))
	if err != nil {
		return err
	}

	for _, item := range registryItems {
		switch item.Type {
		case "combinedsewageoverflow":
			// TODO: exec in goroutine?
			cso := combinedsewageoverflow.New(a.storage, a.msgCtx)
			err := cso.Handle(ctx, &msg, item.Options...)
			if err != nil {
				// TODO: log error and continue?
				return err
			}
		case "sumppump":
			sp := sumppump.New(a.storage, a.msgCtx, item.FnID)
			err := sp.Handle(ctx, &msg, item.Options...)
			if err != nil {
				return err
			}
		default:
			log.Debug("unknown function", "name", item.Type)
		}
	}

	return nil
}
