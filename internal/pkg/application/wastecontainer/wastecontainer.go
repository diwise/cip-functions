package wastecontainer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/senml"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

func RegisterMessageHandlers(msgCtx messaging.MsgContext, tc things.Client, s storage.Storage) error {
	var err error
	var errs []error

	err = msgCtx.RegisterTopicMessageHandlerWithFilter("function.updated", newLevelMessageHandler(msgCtx, tc, s), func(m messaging.Message) bool {
		return strings.HasPrefix(m.ContentType(), "application/vnd.diwise.level")
	})
	if err != nil {
		errs = append(errs, err)
	}

	err = msgCtx.RegisterTopicMessageHandlerWithFilter("message.accepted", newTemperatureMessageHandler(msgCtx, tc, s), func(m messaging.Message) bool {
		return strings.HasPrefix(m.ContentType(), "application/vnd.oma.lwm2m.ext.3303")
	})
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type WasteContainer struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Level        float64   `json:"level"`
	Temperature  float64   `json:"temperature"`
	DateObserved time.Time `json:"dateObserved"`
}

func (wc WasteContainer) TopicName() string {
	return "cip-function.updated"
}

func (wc WasteContainer) ContentType() string {
	return "application/vnd.diwise.wastecontainer+json"
}

func (wc WasteContainer) Body() []byte {
	b, _ := json.Marshal(wc)
	return b
}

func (wc *WasteContainer) Handle(ctx context.Context, itm messaging.IncomingTopicMessage) error {
	var err error

	m := msg{}
	err = json.Unmarshal(itm.Body(), &m)
	if err != nil {
		return err
	}

	if m.Level != nil {
		wc.Level = *m.Level.Percent
		wc.DateObserved = time.Now().UTC()
	}

	if m.Pack == nil {
		return nil
	}

	if t, ok := m.Pack.GetValue(senml.FindByName("5700")); ok {
		wc.Temperature = t
	}

	if ts, ok := m.Pack.GetTime(senml.FindByName("5700")); ok {
		if ts.After(wc.DateObserved) {
			wc.DateObserved = ts
		}
	}

	return nil
}

func newTemperatureMessageHandler(msgCtx messaging.MsgContext, tc things.Client, s storage.Storage) messaging.TopicMessageHandler {
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

		deviceID := strings.Split(r.BaseName, "/")[0]

		things, err := tc.FindRelatedThings(ctx, deviceID)
		if err != nil {
			log.Error("could not query for things", "err", err.Error())
			return
		}

		if len(things) == 0 {
			log.Debug("no related things")
			return
		}

		err = process(ctx, msgCtx, itm, s, things)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			return
		}
	}
}

func newLevelMessageHandler(msgCtx messaging.MsgContext, tc things.Client, s storage.Storage) messaging.TopicMessageHandler {
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

		things, err := tc.FindRelatedThings(ctx, f.ID)
		if err != nil {
			log.Error("could not query for things", "err", err.Error())
			return
		}

		if len(things) == 0 {
			log.Debug("no related things")
			return
		}

		err = process(ctx, msgCtx, itm, s, things)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			return
		}
	}
}

func process(ctx context.Context, msgCtx messaging.MsgContext, itm messaging.IncomingTopicMessage, s storage.Storage, things []things.Thing) error {
	log := logging.GetFromContext(ctx)

	wasteContainerId, hasWasteContainer := containsWasteContainer(things)
	if !hasWasteContainer {
		return fmt.Errorf("contains no wastecontainer")
	}

	wc, err := storage.GetOrDefault(ctx, s, wasteContainerId, WasteContainer{ID: wasteContainerId, Type: "WasteContainer"})
	if err != nil {
		log.Error("could not get or create current state for wastecontainer", "wastecontainer_id", wasteContainerId, "err", err.Error())
		return err
	}

	err = wc.Handle(ctx, itm)
	if err != nil {
		log.Error("could not handle incommig message", "err", err.Error())
		return err
	}

	err = storage.CreateOrUpdate(ctx, s, wc.ID, wc)
	if err != nil {
		log.Error("could not store state", "err", err.Error())
		return err
	}

	err = msgCtx.PublishOnTopic(ctx, wc)
	if err != nil {
		log.Error("could not publish message", "err", err.Error())
		return err
	}

	return nil
}

func containsWasteContainer(ts []things.Thing) (string, bool) {
	idx := slices.IndexFunc(ts, func(t things.Thing) bool {
		return strings.EqualFold(t.Type, "WasteContainer")
	})

	if idx == -1 {
		return "", false
	}

	return ts[idx].Id, true
}

type msg struct {
	ID     *string     `json:"id,omitempty"`
	Tenant *string     `json:"tenant,omitempty"`
	Level  *level      `json:"level,omitempty"`
	Pack   *senml.Pack `json:"pack,omitempty"`
}

type level struct {
	Current float64  `json:"current"`
	Percent *float64 `json:"percent,omitempty"`
}
