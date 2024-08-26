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
	"github.com/diwise/cip-functions/internal/pkg/application/sewer"
	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/application/wastecontainer"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("cip-functions")

type CipFunctionHandler interface {
	messaging.TopicMessage
	Handle(ctx context.Context, itm messaging.IncomingTopicMessage, tc things.Client) (bool, error)
}

type App struct {
	msgCtx       messaging.MsgContext
	thingsClient things.Client
	store        storage.Storage
}

func New(msgCtx messaging.MsgContext, tc things.Client, s storage.Storage) (App, error) {
	app := App{
		msgCtx:       msgCtx,
		thingsClient: tc,
		store:        s,
	}

	return app, app.registerMessageHandlers()
}

func (a App) registerMessageHandlers() error {
	var err error
	var errs []error

	err = a.msgCtx.RegisterTopicMessageHandler("function.updated", newFunctionUpdatedHandler(a))
	if err != nil {
		errs = append(errs, err)
	}

	err = a.msgCtx.RegisterTopicMessageHandler("message.accepted", newMessageAcceptedHandler(a))
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func newMessageAcceptedHandler(app App) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, l *slog.Logger) {
		var err error
		var errs []error

		ctx, span := tracer.Start(ctx, "message.accepted")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, l = o11y.AddTraceIDToLoggerAndStoreInContext(span, l, ctx)

		l.Debug("incoming message", slog.String("body", string(itm.Body())), slog.String("topic_name", itm.TopicName()), slog.String("content_type", itm.ContentType()))

		ctx = logging.NewContextWithLogger(ctx, l, slog.String("uuid", uuid.NewString()))

		if TemperatureMessageFilter(itm) {
			err = handleMessageAcceptedMessage(ctx, app, itm, wastecontainer.WasteContainerFactory, l)
			if err != nil {
				errs = append(errs, err)
			}
		}

		if DistanceMessageFilter(itm) {
			err = handleMessageAcceptedMessage(ctx, app, itm, sewer.SewerFactory, l)
			if err != nil {
				errs = append(errs, err)
			}
		}

		err = errors.Join(errs...)
		if err != nil {
			l.Error("could not handle message.accepted without errors", "err", err.Error())
		}
	}
}

func newFunctionUpdatedHandler(app App) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, l *slog.Logger) {
		var err error
		var errs []error

		ctx, span := tracer.Start(ctx, "function.updated")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, l = o11y.AddTraceIDToLoggerAndStoreInContext(span, l, ctx)

		l.Debug("incoming message", slog.String("body", string(itm.Body())), slog.String("topic_name", itm.TopicName()), slog.String("content_type", itm.ContentType()))

		ctx = logging.NewContextWithLogger(ctx, l, slog.String("uuid", uuid.NewString()))

		if LevelMessageFilter(itm) {
			err = handleFunctionUpdatedMessage(ctx, app, itm, wastecontainer.WasteContainerFactory, l)
			if err != nil {
				errs = append(errs, err)
			}
			err = handleFunctionUpdatedMessage(ctx, app, itm, sewer.SewerFactory, l)
			if err != nil {
				errs = append(errs, err)
			}
		}

		if StopwatchMessageFilter(itm) {
			err = handleFunctionUpdatedMessage(ctx, app, itm, combinedsewageoverflow.CombinedSewageOverflowFactory, l)
			if err != nil {
				errs = append(errs, err)
			}
		}

		if DigitalInputMessageFilter(itm) {
			err = handleFunctionUpdatedMessage(ctx, app, itm, sewagepumpingstation.SewagePumpingStationFactory, l)
			if err != nil {
				errs = append(errs, err)
			}
		}

		err = errors.Join(errs...)
		if err != nil {
			l.Error("could not handle function.updated without errors", "err", err.Error())
		}
	}
}

func handleMessageAcceptedMessage[T CipFunctionHandler](ctx context.Context, app App, itm messaging.IncomingTopicMessage, factoryFn func(id, tenant string) T, log *slog.Logger) error {
	var err error

	m := struct {
		Pack      senml.Pack `json:"pack"`
		Timestamp time.Time  `json:"timestamp"`
	}{}

	err = json.Unmarshal(itm.Body(), &m)
	if err != nil {
		log.Error("unmarshal error", "err", err.Error())
		return err
	}

	r, ok := m.Pack.GetRecord(senml.FindByName("0"))
	if !ok {
		log.Error("package contains no deviceID")
		return err
	}

	deviceID := strings.Split(r.Name, "/")[0]
	if deviceID == "" {
		b, _ := json.Marshal(m)
		log.Error("deviceID is empty")
		log.Debug("deviceID is empty", "message", string(b))
		return err
	}

	log = log.With(slog.String("device_id", deviceID))
	ctx = logging.NewContextWithLogger(ctx, log)

	_, err = processIncomingTopicMessage(ctx, app, deviceID, "Device", itm, factoryFn) // all message.accepted are from a "Device"
	if err != nil {
		log.Error("failed to handle message", "err", err.Error())
		return err
	}

	return nil
}

func handleFunctionUpdatedMessage[T CipFunctionHandler](ctx context.Context, app App, itm messaging.IncomingTopicMessage, factoryFn func(id, tenant string) T, log *slog.Logger) error {
	var err error

	f := struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}{}

	log.Debug("incoming function.updated message", slog.String("body", string(itm.Body())), slog.String("topic_name", itm.TopicName()), slog.String("content_type", itm.ContentType()))

	err = json.Unmarshal(itm.Body(), &f)
	if err != nil {
		log.Error("unmarshal error", "err", err.Error())
		return err
	}

	if f.ID == "" {
		log.Error("ID is empty")
		log.Debug("ID is empty", "message", string(itm.Body()))
		return err
	}

	log = log.With(slog.String("function_id", f.ID), slog.String("function_type", f.Type))
	ctx = logging.NewContextWithLogger(ctx, log)

	_, err = processIncomingTopicMessage(ctx, app, f.ID, f.Type, itm, factoryFn)
	if err != nil {
		log.Error("failed to handle message", "err", err.Error())
		return err
	}

	return nil
}

func processIncomingTopicMessage[T CipFunctionHandler](ctx context.Context, app App, id, type_ string, itm messaging.IncomingTopicMessage, fn func(id, t string) T) (bool, error) {
	log := logging.GetFromContext(ctx)

	thingType := storage.GetTypeName[T]()

	log = log.With(slog.String("thing_type", thingType))
	ctx = logging.NewContextWithLogger(ctx, log)

	theThing, err := app.getRelated(ctx, id, type_, thingType) // type is a function or a device, ex: stopwatch for CombinedSewerOwerflow (thingType)
	if err != nil {
		log.Error("could not find thing to process, no such related thing found on function/device", "err", err.Error())
		return false, err
	}

	log = log.With(slog.String("thing_id", theThing.ID))
	ctx = logging.NewContextWithLogger(ctx, log)

	theThing, err = app.thingsClient.FindByID(ctx, theThing.ID, theThing.Type)
	if err != nil {
		log.Error("could not fetch thing", "err", err.Error())
		return false, err
	}

	tenant := "default"
	if theThing.Tenant != "" {
		log.Debug(fmt.Sprintf("use tenant from related object (%s)", theThing.Tenant))
		tenant = theThing.Tenant
	} else {
		log.Debug(fmt.Sprintf("no tenant information on related thing %s (%s)", theThing.ID, theThing.Tenant))
		b, _ := json.Marshal(theThing)
		log.Debug(fmt.Sprintf("thing: \"%s\"", string(b)))
	}

	state, err := storage.GetOrDefault(ctx, app.store, theThing.ID, fn(theThing.ID, tenant))
	if err != nil {
		log.Error("could not get or create current state", "err", err.Error())
		return false, err
	}

	change, err := state.Handle(ctx, itm, app.thingsClient)
	if err != nil {
		log.Error("could not handle incomig message", "err", err.Error())
		return false, err
	}

	log.Debug(fmt.Sprintf("processed incomming message %s, change is %t", itm.ContentType(), change))

	if !change {
		return false, nil
	}

	err = storage.CreateOrUpdate(ctx, app.store, theThing.ID, state)
	if err != nil {
		log.Error("could not store state", "err", err.Error())
		return change, nil
	}

	err = app.msgCtx.PublishOnTopic(ctx, state)
	if err != nil {
		log.Error("could not publish message", "err", err.Error())
		return change, err
	}

	log.Debug("handled incoming message", slog.String("in", itm.ContentType()), slog.String("out", state.ContentType()))

	return change, nil
}

var ErrNoRelatedThingFound = fmt.Errorf("no related thing found")

// id - function/device id 
// typeName - function type (stopwatch, level ...) or "Device"
// relatedType - type of related thing, i.e. CombinedSewerOverflow, Sewer, WasteContainer ...
func (a App) getRelated(ctx context.Context, id, typeName, relatedType string) (things.Thing, error) {
	log := logging.GetFromContext(ctx)

	// fetch "function"/device and returns .Included
	ts, err := a.thingsClient.FindRelatedThings(ctx, id, typeName)
	if err != nil {
		if errors.Is(err, things.ErrThingNotFound) {
			return things.Thing{}, ErrNoRelatedThingFound
		}

		log.Error(fmt.Sprintf("failed to get related things - %s", err.Error()))
		return things.Thing{}, err
	}

	if len(ts) == 0 {
		return things.Thing{}, ErrNoRelatedThingFound
	}

	// gets the first thing of correct type.
	idx := slices.IndexFunc(ts, func(t things.Thing) bool {
		return strings.EqualFold(t.Type, relatedType)
	})

	if idx == -1 {
		return things.Thing{}, ErrNoRelatedThingFound
	}

	return ts[idx], nil
}
