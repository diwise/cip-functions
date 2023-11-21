package functions

import (
	"context"

	"github.com/diwise/cip-functions/internal/pkg/application/functions/combinedsewageoverflow"
	"github.com/diwise/cip-functions/internal/pkg/application/functions/options"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

//go:generate moq -rm -out fn_mock.go . Fn
type Fn interface {
	ID() string
	Type() string
	Handle(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error
}

type fnImpl struct {
	ID_   string `json:"id"`
	Type_ string `json:"type"`

	SewageOverflow combinedsewageoverflow.SewageOverflow `json:"sewageOverflow,omitempty"`

	handle func(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error
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
func (fn *fnImpl) Handle(ctx context.Context, msg *events.FunctionUpdated, storage database.Storage, msgCtx messaging.MsgContext, opts ...options.Option) error {
	return fn.handle(ctx, msg, storage, msgCtx, opts...)
}


