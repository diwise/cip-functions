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
	Tenant       string    `json:"tenant"`
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

func (wc *WasteContainer) Handle(ctx context.Context, itm messaging.IncomingTopicMessage) (bool, error) {
	var err error
	changed := false

	m := struct {
		ID     *string     `json:"id,omitempty"`
		Tenant *string     `json:"tenant,omitempty"`
		Pack   *senml.Pack `json:"pack,omitempty"`
		Level  *struct {
			Current float64  `json:"current"`
			Percent *float64 `json:"percent,omitempty"`
		} `json:"level,omitempty"`
	}{}

	err = json.Unmarshal(itm.Body(), &m)
	if err != nil {
		return changed, err
	}

	if m.Level != nil && m.Level.Percent != nil {
		wc.Level = *m.Level.Percent

		if wc.DateObserved.IsZero() {
			wc.DateObserved = time.Now().UTC()			
		}
		
		changed = true
	}

	if m.Pack == nil {
		return changed, nil
	}

	if t, ok := m.Pack.GetValue(senml.FindByName("5700")); ok {
		wc.Temperature = t
		changed = true
	}

	if ts, ok := m.Pack.GetTime(senml.FindByName("5700")); ok {
		if ts.After(wc.DateObserved) {
			wc.DateObserved = ts
			changed = true
		}
	}

	if wc.DateObserved.IsZero() {
		wc.DateObserved = time.Now().UTC()
		changed = true
	}

	return changed, nil
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

		deviceID := strings.Split(r.Name, "/")[0]
		if deviceID == "" {
			b, _ := json.Marshal(m)
			log.Error("deviceID is empty", "message", string(b))
			return
		}

		log.Debug("process temperature message for device", slog.String("device_id", deviceID))

		err = process(ctx, msgCtx, itm, s, tc, deviceID)
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

		err = process(ctx, msgCtx, itm, s, tc, f.ID)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			return
		}
	}
}

func process(ctx context.Context, msgCtx messaging.MsgContext, itm messaging.IncomingTopicMessage, s storage.Storage, tc things.Client, id string) error {
	log := logging.GetFromContext(ctx)

	wasteContainer, err := getRelatedWasteContainer(ctx, tc, id)
	if err != nil {
		return err
	}

	tenant := "default"
	if wasteContainer.Tenant != nil {
		tenant = *wasteContainer.Tenant
	}

	wc, err := storage.GetOrDefault(ctx, s, wasteContainer.Id, WasteContainer{ID: wasteContainer.Id, Type: "WasteContainer", Tenant: tenant})
	if err != nil {
		log.Error("could not get or create current state for waste container", "wastecontainer_id", wasteContainer.Id, "err", err.Error())
		return err
	}

	changed, err := wc.Handle(ctx, itm)
	if err != nil {
		log.Error("could not handle incomig message", "err", err.Error())
		return err
	}

	if !changed {
		return nil
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

var ErrContainsNoWasteContainer = fmt.Errorf("contains no waste container")

func getRelatedWasteContainer(ctx context.Context, tc things.Client, id string) (things.Thing, error) {
	ths, err := tc.FindRelatedThings(ctx, id)
	if err != nil {
		return things.Thing{}, err
	}

	wasteContainerId, hasWasteContainer := containsWasteContainer(ths)
	if !hasWasteContainer {
		return things.Thing{}, ErrContainsNoWasteContainer
	}

	return tc.FindByID(ctx, wasteContainerId)
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
