package application

import (
	"strings"

	"github.com/diwise/messaging-golang/pkg/messaging"
)

var LevelMessageFilter = func(m messaging.Message) bool {
	return strings.HasPrefix(m.ContentType(), "application/vnd.diwise.level")
}

var StopwatchMessageFilter = func(m messaging.Message) bool {
	return strings.HasPrefix(m.ContentType(), "application/vnd.diwise.stopwatch")
}

var DigitalInputMessageFilter = func(m messaging.Message) bool {
	return strings.HasPrefix(m.ContentType(), "application/vnd.diwise.digitalinput")
}

var TemperatureMessageFilter = func(m messaging.Message) bool {
	return strings.HasPrefix(m.ContentType(), "application/vnd.oma.lwm2m.ext.3303")
}

var DistanceMessageFilter = func(m messaging.Message) bool {
	return strings.HasPrefix(m.ContentType(), "application/vnd.oma.lwm2m.ext.3330")
}
