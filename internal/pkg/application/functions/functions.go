package functions

import (
	"context"

	"github.com/diwise/cip-functions/pkg/messaging/events"
)

type Function interface {
	Handle(context.Context, *events.FunctionUpdated) error
}
