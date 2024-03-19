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
	"github.com/diwise/cip-functions/internal/pkg/application/sewagepumpingstation"
	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/application/wastecontainer"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("cip-functions")

type CipFunctionHandler interface {
	messaging.TopicMessage
	Handle(ctx context.Context, itm messaging.IncomingTopicMessage) (bool, error)
}

type App struct {
	msgCtx       messaging.MsgContext
	thingsClient things.Client
	store        storage.Storage
}

func New(msgCtx messaging.MsgContext, tc things.Client, s storage.Storage) App {
	return App{
		msgCtx:       msgCtx,
		thingsClient: tc,
		store:        s,
	}
}

const (
	FunctionUpdatedTopic string = "function.updated"
	MessageAcceptedTopic string = "message.accepted"
)

func RegisterMessageHandlers(app App) error {
	var err error
	var errs []error

	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(FunctionUpdatedTopic, newFunctionUpdatedHandler(app, wastecontainer.WasteContainerFactory), LevelMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}
	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(MessageAcceptedTopic, newMessageAcceptedMessageHandler(app, wastecontainer.WasteContainerFactory), TemperatureMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}

	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(FunctionUpdatedTopic, newFunctionUpdatedHandler(app, combinedsewageoverflow.CombinedSewageOverflowFactory), StopwatchMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}

	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(FunctionUpdatedTopic, newFunctionUpdatedHandler(app, sewagepumpingstation.SewagePumpingStationFactory), DigitalInputMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func newMessageAcceptedMessageHandler[T CipFunctionHandler](app App, fn func(id, tenant string) T) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, log *slog.Logger) {
		var err error

		ctx, span := tracer.Start(ctx, "message.accepted")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

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
			log.Error("deviceID is empty")
			log.Debug("deviceID is empty", "message", string(b))
			return
		}

		_, err = process(ctx, app, deviceID, itm, fn)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			return
		}
	}
}

func newFunctionUpdatedHandler[T CipFunctionHandler](app App, fn func(id, tenant string) T) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, log *slog.Logger) {
		var err error

		ctx, span := tracer.Start(ctx, "function.updated")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		f := struct {
			ID string `json:"id,omitempty"`
		}{}

		err = json.Unmarshal(itm.Body(), &f)
		if err != nil {
			log.Error("unmarshal error", "err", err.Error())
			return
		}

		if f.ID == "" {
			log.Error("functionID is empty")
			log.Debug("functionID is empty", "message", string(itm.Body()))
			return
		}

		_, err = process(ctx, app, f.ID, itm, fn)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			return
		}
	}
}

func process[T CipFunctionHandler](ctx context.Context, app App, id string, itm messaging.IncomingTopicMessage, fn func(id, t string) T) (bool, error) {
	log := logging.GetFromContext(ctx)

	rel, err := getRelatedThing[T](ctx, app, id)
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

func (a App) getRelated(ctx context.Context, id, typeName string) (things.Thing, error) {
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

func getRelatedThing[T any](ctx context.Context, app App, id string) (things.Thing, error) {
	typeName := storage.GetTypeName[T]()
	return app.getRelated(ctx, id, typeName)
}
