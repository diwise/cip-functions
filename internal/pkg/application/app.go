package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
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

	_, err = processIncomingTopicMessage(ctx, app, deviceID, itm, factoryFn)
	if err != nil {
		log.Error("failed to handle message", "err", err.Error())
		return err
	}

	return nil
}

func handleFunctionUpdatedMessage[T CipFunctionHandler](ctx context.Context, app App, itm messaging.IncomingTopicMessage, factoryFn func(id, tenant string) T, log *slog.Logger) error {
	var err error

	f := struct {
		ID string `json:"id,omitempty"`
	}{}

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

	log = log.With(slog.String("function_id", f.ID))
	ctx = logging.NewContextWithLogger(ctx, log)

	_, err = processIncomingTopicMessage(ctx, app, f.ID, itm, factoryFn)
	if err != nil {
		log.Error("failed to handle message", "err", err.Error())
		return err
	}

	return nil
}

var mu sync.Mutex

func processIncomingTopicMessage[T CipFunctionHandler](ctx context.Context, app App, id string, itm messaging.IncomingTopicMessage, fn func(id, t string) T) (bool, error) {
	mu.Lock()
	defer mu.Unlock()

	log := logging.GetFromContext(ctx)
	log = log.With(slog.String("type", storage.GetTypeName[T]()))
	ctx = logging.NewContextWithLogger(ctx, log)

	rel, ok, err := getRelatedThing[T](ctx, app, id)
	if err != nil {
		log.Error("could not fetch related thing", "err", err.Error())
		return false, err
	}

	if !ok {
		return false, nil
	}

	log = log.With(slog.String("thing_id", rel.Id))
	ctx = logging.NewContextWithLogger(ctx, log)

	tenant := "default"
	if rel.Tenant != "" {
		tenant = rel.Tenant
	}

	state, err := storage.GetOrDefault(ctx, app.store, rel.Id, fn(rel.Id, tenant))
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

	err = storage.CreateOrUpdate(ctx, app.store, rel.Id, state)
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

func getRelatedThing[T any](ctx context.Context, app App, id string) (things.Thing, bool, error) {
	typeName := storage.GetTypeName[T]()

	t, err := app.getRelated(ctx, id, typeName)
	if err != nil {
		if errors.Is(err, ErrNoRelatedThingFound) {
			return things.Thing{}, false, nil
		}
		return things.Thing{}, false, err
	}

	return t, true, nil
}

func (a App) getRelated(ctx context.Context, id, typeName string) (things.Thing, error) {
	log := logging.GetFromContext(ctx)

	ts, err := a.thingsClient.FindRelatedThings(ctx, id)
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

	idx := slices.IndexFunc(ts, func(t things.Thing) bool {
		return strings.EqualFold(t.Type, typeName)
	})

	if idx == -1 {
		return things.Thing{}, ErrNoRelatedThingFound
	}

	return ts[idx], nil
}
