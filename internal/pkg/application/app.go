package application

import (
	"errors"

	"github.com/diwise/cip-functions/internal/pkg/application/combinedsewageoverflow"
	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/application/wastecontainer"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
)

func RegisterMessageHandlers(msgCtx messaging.MsgContext, tc things.Client, s storage.Storage) error {
	var err error
	var errs []error

	err = wastecontainer.RegisterMessageHandlers(msgCtx, tc, s)
	if err != nil {
		errs = append(errs, err)
	}

	err = combinedsewageoverflow.RegisterMessageHandlers(msgCtx, tc, s)
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}