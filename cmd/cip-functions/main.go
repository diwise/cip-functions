package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/diwise/cip-functions/internal/pkg/application"
	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage/database"
	api "github.com/diwise/cip-functions/internal/pkg/presentation"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

const serviceName string = "cip-functions"

var tracer = otel.Tracer(serviceName)
var functionsConfigPath string

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, _, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	flag.StringVar(&functionsConfigPath, "functions", "/opt/diwise/config/cip-functions.csv", "configuration file for functions")
	flag.Parse()

	var err error

	msgCtx := createMessagingContextOrDie(ctx)
	defer msgCtx.Close()

	storage := createDatabaseConnectionOrDie(ctx)
	thingsClient := createThingsClientOrDie(ctx)

	err = application.RegisterMessageHandlers(msgCtx, thingsClient, storage)
	if err != nil {
		fatal(ctx, "failed to register handlers", err)
	}

	var configFile *os.File

	if functionsConfigPath != "" {
		configFile, err = os.Open(functionsConfigPath)
		if err != nil {
			fatal(ctx, "failed to open functions config file", err)
		}
		defer configFile.Close()
	}

	_, api_, err := initialize(ctx, msgCtx, configFile, storage)
	if err != nil {
		fatal(ctx, "initialization failed", err)
	}

	servicePort := env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080")
	err = http.ListenAndServe(":"+servicePort, api_.Router())
	if err != nil {
		fatal(ctx, "failed to start request router", err)
	}
}

func createMessagingContextOrDie(ctx context.Context) messaging.MsgContext {
	logger := logging.GetFromContext(ctx)

	config := messaging.LoadConfiguration(ctx, serviceName, logger)
	messenger, err := messaging.Initialize(ctx, config)
	if err != nil {
		fatal(ctx, "failed to init messaging", err)
	}

	messenger.Start()
	logger.Info("starting messaging service")

	return messenger
}

func createDatabaseConnectionOrDie(ctx context.Context) storage.Storage {
	storage, err := database.Connect(ctx, database.LoadConfiguration(ctx))
	if err != nil {
		fatal(ctx, "database connect failed", err)
	}
	err = storage.Initialize(ctx)
	if err != nil {
		fatal(ctx, "database initialize failed", err)
	}
	return storage
}

func createThingsClientOrDie(ctx context.Context) *things.ClientImpl {
	tokenUrl := env.GetVariableOrDie(ctx, "OAUTH2_TOKEN_URL", "")
	clientId := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_ID", "")
	secret := env.GetVariableOrDie(ctx, "OAUTH2_CLIENT_SECRET", "")
	thingsUrl := env.GetVariableOrDefault(ctx, "THINGS_URL", "http://iot-things:8080")

	c, err := things.NewClient(ctx, thingsUrl, tokenUrl, clientId, secret)
	if err != nil {
		fatal(ctx, "", err)
	}

	return c
}

func initialize(ctx context.Context, msgctx messaging.MsgContext, fconfig io.Reader, storage storage.Storage) (application.App, api.API, error) {
	fnRegistry, err := functions.NewRegistry(ctx, fconfig)
	if err != nil {
		return nil, nil, err
	}

	app := application.New(storage, msgctx, fnRegistry)

	routingKey := "function.updated"
	msgctx.RegisterTopicMessageHandler(routingKey, newTopicMessageHandler(app))

	return app, api.New(ctx), nil
}

func newTopicMessageHandler(app application.App) messaging.TopicMessageHandler {
	return func(ctx context.Context, itm messaging.IncomingTopicMessage, l *slog.Logger) {
		var err error

		ctx, span := tracer.Start(ctx, "receive-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, l = o11y.AddTraceIDToLoggerAndStoreInContext(span, l, ctx)

		l.Debug("received message", "body", string(itm.Body()))

		evt := events.FunctionUpdated{}

		err = json.Unmarshal(itm.Body(), &evt)
		if err != nil {
			l.Error("unable to unmarshal incoming message", "err", err.Error())
			return
		}

		l = l.With(slog.String("function_id", evt.ID))
		ctx = logging.NewContextWithLogger(ctx, l)

		err = app.FunctionUpdated(ctx, evt)
		if err != nil {
			l.Error("failed to handle message", "err", err.Error())
		}
	}
}

func fatal(ctx context.Context, msg string, err error) {
	logger := logging.GetFromContext(ctx)
	logger.Error(msg, "err", err.Error())
	os.Exit(1)
}
