package wastecontainer

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	client "github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type WasteContainerState struct {
	ID string `json:"id"`
	Type string `json:"type"`
}

func (sp WasteContainerState) TopicName() string {
	return "cip-function.updated"
}

func (sp WasteContainerState) ContentType() string {
	return "application/vnd.diwise.wastecontainer+json"
}

func (sp WasteContainerState) Body() []byte {
	b, _ := json.Marshal(sp)
	return b
}

func (wc *WasteContainerState) Handle(ctx context.Context, itm messaging.IncomingTopicMessage) error {

	//TODO: read level info
	//		update state

	return nil
}

func Register(msgCtx messaging.MsgContext, tc client.ThingsClient, s storage.Storage) error {
	return msgCtx.RegisterTopicMessageHandlerWithFilter("function.updated", newLevelMessageHandler(msgCtx, tc, s), func(m messaging.Message) bool {
		return strings.HasPrefix(m.ContentType(), "application/vnd.diwise.level")
	})
}

func newLevelMessageHandler(msgCtx messaging.MsgContext, tc client.ThingsClient, s storage.Storage) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, l *slog.Logger) {
		var err error

		log := logging.GetFromContext(ctx)

		f := struct {
			ID      *string `json:"id,omitempty"`
			Type    *string `json:"type,omitempty"`
			SubType *string `json:"subType,omitempty"`
		}{}

		err = json.Unmarshal(itm.Body(), &f)
		if err != nil {
			log.Error("unmarshal error", "err", err.Error())
			return
		}

		if f.ID == nil {
			log.Error("message contains no ID")
		}

		things, err := tc.FindRelatedThings(ctx, *f.ID)
		if err != nil {
			log.Error("could not query for things", "err", err.Error())
			return
		}

		if len(things) == 0 {
			log.Debug("no related things")
			return
		}

		wasteContainerId, hasWasteContainer := containsWasteContainer(things)
		if !hasWasteContainer {
			return
		}

		state, err := storage.GetOrDefault(ctx, s, wasteContainerId, WasteContainerState{ID: wasteContainerId, Type: "WasteContainer"})
		if err != nil {
			log.Error("could not get current state for wastecontainer", "wastecontainer_id", wasteContainerId, "err", err.Error())
			return
		}

		err = state.Handle(ctx, itm)
		if err != nil {
			log.Error("could not handle incommig message", "err", err.Error())
			return
		}

		err = storage.CreateOrUpdate(ctx, s, state.ID, state)
		if err != nil {
			log.Error("could not store state", "err", err.Error())
			return
		}

		err = msgCtx.PublishOnTopic(ctx, state)
		if err != nil {
			log.Error("could not publish message", "err", err.Error())
			return
		}
	}
}

func containsWasteContainer(things []client.Thing) (string, bool) {
	for _, t := range things {
		if strings.EqualFold(t.Type, "WasteContainer") {
			return t.Id, true
		}
	}
	return "", false
}