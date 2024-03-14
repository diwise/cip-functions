package combinedsewageoverflow

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/google/uuid"
)

func RegisterMessageHandlers(msgCtx messaging.MsgContext, tc things.Client, s storage.Storage) error {
	return msgCtx.RegisterTopicMessageHandlerWithFilter("function.updated", newStopwatchMessageHandler(msgCtx, tc, s), func(m messaging.Message) bool {
		return strings.HasPrefix(m.ContentType(), "application/vnd.diwise.stopwatch")
	})
}

func newStopwatchMessageHandler(msgCtx messaging.MsgContext, tc things.Client, s storage.Storage) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, log *slog.Logger) {
		var err error

		stopwatch := struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			SubType string `json:"subtype"`
		}{}

		err = json.Unmarshal(itm.Body(), &stopwatch)
		if err != nil {
			log.Error("unmarshal error", "err", err.Error())
			return
		}

		err = process(ctx, msgCtx, itm, s, tc, stopwatch.ID)
		if err != nil {
			log.Error("failed to handle message", "err", err.Error())
			return
		}
	}
}

type CombinedSewageOverflow struct {
	ID             string        `json:"id"`
	Type           string        `json:"type"`
	Overflows      []Overflow    `json:"overflow"`
	DateObserved   time.Time     `json:"dateObserved"`
	CumulativeTime time.Duration `json:"cumulativeTime"`
	Tenant         string        `json:"tenant"`
}

type Overflow struct {
	ID        string        `json:"id"`
	State     bool          `json:"state"`
	StartTime time.Time     `json:"startTime"`
	StopTime  *time.Time    `json:"stopTime"`
	Duration  time.Duration `json:"duration"`
}

type functionUpdated struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	SubType   string    `json:"subtype"`
	Stopwatch stopwatch `json:"stopwatch"`
}

type stopwatch struct {
	StartTime time.Time      `json:"startTime"`
	StopTime  *time.Time     `json:"stopTime,omitempty"`
	Duration  *time.Duration `json:"duration,omitempty"`

	State bool  `json:"state"`
	Count int32 `json:"count"`

	CumulativeTime time.Duration `json:"cumulativeTime"`
}

func (cso CombinedSewageOverflow) TopicName() string {
	return "cip-function.updated"
}

func (cso CombinedSewageOverflow) ContentType() string {
	return "application/vnd.diwise.combinedsewageoverflow+json"
}

func (cso CombinedSewageOverflow) Body() []byte {
	b, _ := json.Marshal(cso)
	return b
}

func getStopwatch(itm messaging.IncomingTopicMessage) (*stopwatch, error) {
	stopwatch := functionUpdated{}

	err := json.Unmarshal(itm.Body(), &stopwatch)
	if err != nil {
		return nil, err
	}

	return &stopwatch.Stopwatch, nil
}

func (cso *CombinedSewageOverflow) Handle(ctx context.Context, itm messaging.IncomingTopicMessage) (bool, error) {
	changed := false

	sw, err := getStopwatch(itm)
	if err != nil {
		return changed, err
	}

	overflow := getOrCreateOverflowForStartTime(cso, sw.StartTime)

	if overflow.StopTime != nil {
		return changed, fmt.Errorf("current overflow already ended")
	}

	if overflow.State != sw.State {
		if sw.State {
			if sw.Duration != nil {
				overflow.Duration = *sw.Duration
			}
		}

		if !sw.State {
			overflow.StopTime = sw.StopTime

			if sw.Duration != nil {
				overflow.Duration = *sw.Duration
			}

			if sw.Duration == nil {
				overflow.Duration = overflow.StopTime.Sub(overflow.StartTime)
			}
		}

		cumulativeTime := 0
		for _, o := range cso.Overflows {
			cumulativeTime += int(o.Duration)
		}

		cso.CumulativeTime = time.Duration(cumulativeTime)
		overflow.State = sw.State

		changed = true
	}

	if cso.DateObserved.IsZero() || changed {
		cso.DateObserved = time.Now().UTC()
	}

	n := 10
	if len(cso.Overflows) < n {
		n = len(cso.Overflows)
	}

	cso.Overflows = cso.Overflows[:n]

	return changed, nil
}

func getOrCreateOverflowForStartTime(cso *CombinedSewageOverflow, t time.Time) *Overflow {
	overflowID := deterministicUUID(t)
	idx := slices.IndexFunc(cso.Overflows, func(o Overflow) bool {
		return o.ID == overflowID
	})
	if idx == -1 {
		cso.Overflows = append(cso.Overflows, Overflow{ID: overflowID, StartTime: t})
		idx = len(cso.Overflows) - 1
	}
	return &cso.Overflows[idx]
}

func deterministicUUID(t time.Time) string {
	str := fmt.Sprintf("%d", t.UnixNano())

	md5hash := md5.New()
	md5hash.Write([]byte(str))
	md5string := hex.EncodeToString(md5hash.Sum(nil))

	unique, err := uuid.FromBytes([]byte(md5string[0:16]))
	if err != nil {
		return uuid.New().String()
	}

	return unique.String()
}

func process(ctx context.Context, msgCtx messaging.MsgContext, itm messaging.IncomingTopicMessage, s storage.Storage, tc things.Client, id string) error {
	log := logging.GetFromContext(ctx)

	combinedSewageOverflow, err := getRelatedCombinedSewageOverflow(ctx, tc, id)
	if err != nil {
		return err
	}

	tenant := "default"
	if combinedSewageOverflow.Tenant != nil {
		tenant = *combinedSewageOverflow.Tenant
	}

	cso, err := storage.GetOrDefault(ctx, s, combinedSewageOverflow.Id, CombinedSewageOverflow{ID: combinedSewageOverflow.Id, Type: "CombinedSewageOverflow", Tenant: tenant})
	if err != nil {
		log.Error("could not get or create current state for combined sewage overflow", "combinedsewageoverflow_id", combinedSewageOverflow.Id, "err", err.Error())
		return err
	}

	changed, err := cso.Handle(ctx, itm)
	if err != nil {
		log.Error("could not handle incomig message", "err", err.Error())
		return err
	}

	if !changed {
		return nil
	}

	err = storage.CreateOrUpdate(ctx, s, cso.ID, cso)
	if err != nil {
		log.Error("could not store state", "err", err.Error())
		return err
	}

	err = msgCtx.PublishOnTopic(ctx, cso)
	if err != nil {
		log.Error("could not publish message", "err", err.Error())
		return err
	}

	return nil
}

var ErrContainsNoCombinedSewageOverflow = fmt.Errorf("contains no combined sewage overflow")

func getRelatedCombinedSewageOverflow(ctx context.Context, tc things.Client, id string) (things.Thing, error) {
	ths, err := tc.FindRelatedThings(ctx, id)
	if err != nil {
		return things.Thing{}, err
	}

	combinedSewageOverflowID, hasCombinedSewageOverflow := containsCombinedSewageOverflow(ths)
	if !hasCombinedSewageOverflow {
		return things.Thing{}, ErrContainsNoCombinedSewageOverflow
	}

	return tc.FindByID(ctx, combinedSewageOverflowID)
}

func containsCombinedSewageOverflow(ts []things.Thing) (string, bool) {
	idx := slices.IndexFunc(ts, func(t things.Thing) bool {
		return strings.EqualFold(t.Type, "CombinedSewageOverflow")
	})

	if idx == -1 {
		return "", false
	}

	return ts[idx].Id, true
}
