package wastecontainer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	client "github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/farshidtz/senml/v2"
)

type WasteContainer struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Level        float64   `json:"level"`
	Temperature  float64   `json:"temperature"`
	DateObserved time.Time `json:"dateObserved"`
}

func (sp WasteContainer) TopicName() string {
	return "cip-function.updated"
}

func (sp WasteContainer) ContentType() string {
	return "application/vnd.diwise.wastecontainer+json"
}

func (sp WasteContainer) Body() []byte {
	b, _ := json.Marshal(sp)
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

	temperature := func(p senml.Pack) (float64, bool) {
		for _, r := range p {
			if strings.EqualFold(r.Name, "5700") {
				if r.Value != nil {
					return *r.Value, true
				}
			}
		}
		return 0, false
	}

	timestamp := func(p senml.Pack) (time.Time, bool) {
		c := p.Clone()
		c.Normalize()
		for _, r := range c {
			if strings.HasSuffix(r.Name, "/5700") {
				return time.Unix(int64(r.Time),0), true
			}
		}
		return time.Time{}, false
	}

	if m.Pack != nil {
		if t, ok := temperature(*m.Pack); ok {
			wc.Temperature = t
		}
		if ts, ok := timestamp(*m.Pack); ok {
			if ts.After(wc.DateObserved) {
				wc.DateObserved = ts
			}
		}
	}

	return nil
}

func RegisterMessageHandlers(msgCtx messaging.MsgContext, tc client.ThingsClient, s storage.Storage) error {
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

func newTemperatureMessageHandler(msgCtx messaging.MsgContext, tc client.ThingsClient, s storage.Storage) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, l *slog.Logger) {
		var err error
		log := logging.GetFromContext(ctx)

		m := struct {
			Pack      senml.Pack `json:"pack"`
			Timestamp time.Time  `json:"timestamp"`
		}{}

		deviceID := func(p senml.Pack) string {
			if p[0].Name != "0" {
				return ""
			}
			parts := strings.Split(p[0].BaseName, "/")

			return parts[0]
		}

		err = json.Unmarshal(itm.Body(), &m)
		if err != nil {
			log.Error("unmarshal error", "err", err.Error())
			return
		}

		if deviceID(m.Pack) == "" {
			log.Error("pack contains no deviceID")
			return
		}

		things, err := tc.FindRelatedThings(ctx, deviceID(m.Pack))
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

func newLevelMessageHandler(msgCtx messaging.MsgContext, tc client.ThingsClient, s storage.Storage) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, l *slog.Logger) {
		var err error

		log := logging.GetFromContext(ctx)

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

func process(ctx context.Context, msgCtx messaging.MsgContext, itm messaging.IncomingTopicMessage, s storage.Storage, things []client.Thing) error {
	log := logging.GetFromContext(ctx)

	wasteContainerId, hasWasteContainer := containsWasteContainer(things)
	if !hasWasteContainer {
		return fmt.Errorf("contains no wastecontainer")
	}

	wc, err := storage.GetOrDefault(ctx, s, wasteContainerId, WasteContainer{ID: wasteContainerId, Type: "WasteContainer"})
	if err != nil {
		log.Error("could not get current state for wastecontainer", "wastecontainer_id", wasteContainerId, "err", err.Error())
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

func containsWasteContainer(things []client.Thing) (string, bool) {
	for _, t := range things {
		if strings.EqualFold(t.Type, "WasteContainer") {
			return t.Id, true
		}
	}
	return "", false
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
