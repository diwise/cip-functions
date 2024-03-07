package application

import (
	"context"
	"errors"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
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
	storage    storage.Storage
	msgCtx     messaging.MsgContext
}

func New(s storage.Storage, m messaging.MsgContext, r functions.Registry) App {
	return &app{
		fnRegistry: r,
		storage:    s,
		msgCtx:     m,
	}
}

func (a *app) FunctionUpdated(ctx context.Context, msg events.FunctionUpdated) error {
	log := logging.GetFromContext(ctx)
	log.Info("inside function updated", "id", msg.ID, "type", msg.Type)

	registryItems, err := a.fnRegistry.Find(ctx, functions.FindByFunctionID(msg.ID))
	if err != nil {
		return err
	}
	log.Info("found registry items", "number of items", len(registryItems))

	var errs []error

	for _, item := range registryItems {
		log.Info("looking through registry items", "item", item)
		err := item.Fn.Handle(ctx, &msg, a.storage, a.msgCtx, item.Options...)
		if err != nil {
			log.Error("failed to handle message", "function_id", item.FnID, "type", item.Type, "err", err.Error())
			errs = append(errs, err)
		}

	}

	return errors.Join(errs...)
}
