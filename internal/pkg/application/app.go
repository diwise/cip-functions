package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/combinedsewageoverflow"
	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/application/wastecontainer"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Fnct interface {
	messaging.TopicMessage
	Handle(ctx context.Context, itm messaging.IncomingTopicMessage) (bool, error)
}

type Application struct {
	msgCtx       messaging.MsgContext
	thingsClient things.Client
	store        storage.Storage
}

func NewApplication(msgCtx messaging.MsgContext, tc things.Client, s storage.Storage) Application {
	return Application{
		msgCtx:       msgCtx,
		thingsClient: tc,
		store:        s,
	}
}

var LevelMessageFilter = func(m messaging.Message) bool {
	return strings.HasPrefix(m.ContentType(), "application/vnd.diwise.level")
}

var StopwatchMessageFilter = func(m messaging.Message) bool {
	return strings.HasPrefix(m.ContentType(), "application/vnd.diwise.stopwatch")
}

var TemperatureMessageFilter = func(m messaging.Message) bool {
	return strings.HasPrefix(m.ContentType(), "application/vnd.oma.lwm2m.ext.3303")
}

const (
	FunctionUpdatedTopic string = "function.updated"
	MessageAcceptedTopic string = "message.accepted"
)

func RegisterMessageHandlers(app Application) error {
	var err error
	var errs []error

	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(FunctionUpdatedTopic, NewFunctionUpdatedHandler(app, wastecontainer.WasteContainerFactory), LevelMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}
	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(MessageAcceptedTopic, NewMessageAcceptedMessageHandler(app, wastecontainer.WasteContainerFactory), TemperatureMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}
	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(FunctionUpdatedTopic, NewFunctionUpdatedHandler(app, combinedsewageoverflow.CombinedSewageOverflowFactory), StopwatchMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func NewMessageAcceptedMessageHandler[T Fnct](app Application, fn func(id, tenant string) T) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, log *slog.Logger) {
		var err error

		m := struct {
			Pack      senml.Pack `json:"pack"`
			Timestamp time.Time  `json:"timestamp"`
		}{}

		err = json.Unmarshal(itm.Body(), &m)
		if err != nil {
			log.Error("unmarshal error", "err", err.Error())
			return
		}

		r, ok := m.Pack.GetRecord(senml.FindByName("0"))
		if !ok {
			log.Error("package contains no deviceID")
			return
		}

		deviceID := strings.Split(r.Name, "/")[0]
		if deviceID == "" {
			b, _ := json.Marshal(m)
			log.Error("deviceID is empty", "message", string(b))
			return
		}

		_, err = Process(ctx, app, deviceID, itm, fn)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			return
		}
	}
}

func NewFunctionUpdatedHandler[T Fnct](app Application, fn func(id, tenant string) T) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, log *slog.Logger) {
		var err error

		f := struct {
			ID string `json:"id,omitempty"`
		}{}

		err = json.Unmarshal(itm.Body(), &f)
		if err != nil {
			log.Error("unmarshal error", "err", err.Error())
			return
		}

		_, err = Process(ctx, app, f.ID, itm, fn)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			return
		}
	}
}

func GetRelatedThing[T any](ctx context.Context, app Application, id string) (things.Thing, error) {
	typeName := storage.GetTypeName[T]()
	return app.getRelated(ctx, id, typeName)
}

func (a Application) getRelated(ctx context.Context, id, typeName string) (things.Thing, error) {
	ts, err := a.thingsClient.FindRelatedThings(ctx, id)
	if err != nil {
		return things.Thing{}, err
	}

	idx := slices.IndexFunc(ts, func(t things.Thing) bool {
		return strings.EqualFold(t.Type, typeName)
	})

	if idx == -1 {
		return things.Thing{}, fmt.Errorf("no related thing found")
	}

	return ts[idx], nil
}

func Process[T Fnct](ctx context.Context, app Application, id string, itm messaging.IncomingTopicMessage, fn func(id, t string) T) (bool, error) {
	log := logging.GetFromContext(ctx)

	rel, err := GetRelatedThing[T](ctx, app, id)
	if err != nil {
		return false, err
	}

	tenant := "default"
	if rel.Tenant != nil && *rel.Tenant != "" {
		tenant = *rel.Tenant
	}

	state, err := storage.GetOrDefault(ctx, app.store, rel.Id, fn(rel.Id, tenant))
	if err != nil {
		log.Error("could not get or create current state", "id", rel.Id, "type", rel.Type, "err", err.Error())
		return false, err
	}

	change, err := state.Handle(ctx, itm)
	if err != nil {
		log.Error("could not handle incomig message", "err", err.Error())
		return false, err
	}

	if !change {
		return false, nil
	}

	err = storage.CreateOrUpdate(ctx, app.store, rel.Id, state)
	if err != nil {
		log.Error("could not store state", "err", err.Error())
		return change, nil
	}

	err = app.msgCtx.PublishOnTopic(ctx, state)
	if err != nil {
		log.Error("could not publish message", "err", err.Error())
		return change, nil
	}

	return change, nil
}
