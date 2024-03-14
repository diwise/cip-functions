package functions

import (
	"context"

	"github.com/diwise/cip-functions/internal/pkg/application/functions/options"
	"github.com/diwise/cip-functions/internal/pkg/application/functions/sewagepumpingstation"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

//go:generate moq -rm -out fn_mock.go . Fn
type Fn interface {
	ID() string
	Type() string
	Handle(ctx context.Context, msg *events.FunctionUpdated, store storage.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error
}

type fnImpl struct {
	ID_   string `json:"id"`
	Type_ string `json:"type"`

	IncomingSewagePumpingStation sewagepumpingstation.IncomingSewagePumpingStation `json:"sewagePumpingStation,omitempty"`

	handle func(ctx context.Context, msg *events.FunctionUpdated, store storage.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error
}

func New() Fn {
	return &fnImpl{}
}
func (fn *fnImpl) ID() string {
	return fn.ID_
}
func (fn *fnImpl) Type() string {
	return fn.Type_
}
func (fn *fnImpl) Handle(ctx context.Context, msg *events.FunctionUpdated, store storage.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error {
	return fn.handle(ctx, msg, store, msgCtx, opts...)
}
