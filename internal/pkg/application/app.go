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

	// Waste container
	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(FunctionUpdatedTopic, newFunctionUpdatedHandler(app, wastecontainer.WasteContainerFactory), LevelMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}
	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(MessageAcceptedTopic, newMessageAcceptedMessageHandler(app, wastecontainer.WasteContainerFactory), TemperatureMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}

	// Combined sewer overflow
	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(FunctionUpdatedTopic, newFunctionUpdatedHandler(app, combinedsewageoverflow.CombinedSewageOverflowFactory), StopwatchMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}

	// Sewage pumping station
	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(FunctionUpdatedTopic, newFunctionUpdatedHandler(app, sewagepumpingstation.SewagePumpingStationFactory), DigitalInputMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}

	// Sewer
	err = app.msgCtx.RegisterTopicMessageHandlerWithFilter(FunctionUpdatedTopic, newMessageAcceptedMessageHandler(app, sewer.SewerFactory), DistanceMessageFilter)
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func newMessageAcceptedMessageHandler[T CipFunctionHandler](app App, fn func(id, tenant string) T) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, log *slog.Logger) {
		var err error
		t := storage.GetTypeName[T]()

		ctx, span := tracer.Start(ctx, "message.accepted")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, log = o11y.AddTraceIDToLoggerAndStoreInContext(span, log, ctx)

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

		log = log.With(slog.String("id", deviceID), slog.String("type", t))
		ctx = logging.NewContextWithLogger(ctx, log)

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
		t := storage.GetTypeName[T]()

		ctx, span := tracer.Start(ctx, "function.updated")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
		_, ctx, log = o11y.AddTraceIDToLoggerAndStoreInContext(span, log, ctx)

		f := struct {
			ID string `json:"id,omitempty"`
		}{}

		err = json.Unmarshal(itm.Body(), &f)
		if err != nil {
			log.Error("unmarshal error", "err", err.Error())
			return
		}

		if f.ID == "" {
			log.Error("ID is empty")
			log.Debug("ID is empty", "message", string(itm.Body()))
			return
		}

		log = log.With(slog.String("id", f.ID), slog.String("type", t))
		ctx = logging.NewContextWithLogger(ctx, log)

		_, err = process(ctx, app, f.ID, itm, fn)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			return
		}
	}
}

var mu sync.Mutex

func process[T CipFunctionHandler](ctx context.Context, app App, id string, itm messaging.IncomingTopicMessage, fn func(id, t string) T) (bool, error) {
	mu.Lock()
	defer mu.Unlock()

	log := logging.GetFromContext(ctx)

	rel, ok, err := getRelatedThing[T](ctx, app, id)
	if err != nil {
		log.Error("could not fetch related thing", "err", err.Error())
		return false, err
	}

	if !ok {
		log.Debug("no related thing found")
		return false, nil
	}

	log = log.With(slog.String("rel_id", rel.Id))
	ctx = logging.NewContextWithLogger(ctx, log)

	tenant := "default"
	if rel.Tenant != "" {
		tenant = rel.Tenant
	}

	state, err := storage.GetOrDefault(ctx, app.store, rel.Id, fn(rel.Id, tenant))
	if err != nil {
		log.Error("could not get or create current state", "id", rel.Id, "type", rel.Type, "err", err.Error())
		return false, err
	}

	change, err := state.Handle(ctx, itm, app.thingsClient)
	if err != nil {
		log.Error("could not handle incomig message", "err", err.Error())
		return false, err
	}

	if !change {
		log.Debug("no state change detected")
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

	return change, nil
}

var ErrNoRelatedThingFound = fmt.Errorf("no related thing found")

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
