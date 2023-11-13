package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"os"

	"github.com/diwise/cip-functions/internal/pkg/application"
	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/application/messageprocessor"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/cip-functions/pkg/messaging/events"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
)

const serviceName string = "cip-functions"

var tracer = otel.Tracer(serviceName)
var functionsConfigPath string

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, _, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	flag.StringVar(&functionsConfigPath, "functions", "/opt/diwise/config/functions.csv", "configuration file for functions")
	flag.Parse()

	var err error

	msgCtx := createMessagingContextOrDie(ctx)
	storage := createDatabaseConnectionOrDie(ctx)

	var configFile *os.File

	if functionsConfigPath != "" {
		configFile, err = os.Open(functionsConfigPath)
		if err != nil {
			fatal(ctx, "failed to open functions config file", err)
		}
		defer configFile.Close()
	}

	_, err = initialize(ctx, msgCtx, configFile, storage)
	if err != nil {
		fatal(ctx, "initialization failed", err)
	}
}

func createMessagingContextOrDie(ctx context.Context) messaging.MsgContext {
	logger := logging.GetFromContext(ctx)

	config := messaging.LoadConfiguration(ctx, serviceName, logger)
	messenger, err := messaging.Initialize(ctx, config)
	if err != nil {
		fatal(ctx, "failed to init messaging", err)
	}

	return messenger
}

func createDatabaseConnectionOrDie(ctx context.Context) database.Storage {
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

func initialize(ctx context.Context, msgctx messaging.MsgContext, fconfig io.Reader, storage database.Storage) (application.App, error) {
	msgproc := messageprocessor.NewMessageProcessor()

	functionsRegistry, err := functions.NewRegistry(ctx, fconfig, storage)
	if err != nil {
		return nil, err
	}

	app := application.New(msgproc, functionsRegistry)

	needToDecideThis := "application/json"
	msgctx.RegisterCommandHandler(needToDecideThis, newCommandHandler(msgctx, app))

	routingKey := "message.accepted"
	msgctx.RegisterTopicMessageHandler(routingKey, newTopicMessageHandler(msgctx, app))

	return app, nil
}

func newCommandHandler(messenger messaging.MsgContext, app application.App) messaging.CommandHandler {
	return func(ctx context.Context, wrapper messaging.CommandMessageWrapper, logger *slog.Logger) error {
		var err error

		ctx, span := tracer.Start(ctx, "receive-command")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		evt := events.MessageReceived{}
		err = json.Unmarshal(wrapper.Body(), &evt)
		if err != nil {
			logger.Error("failed to decode message from json", "err", err.Error())
			return err
		}

		logger = logger.With(slog.String("function_id", evt.FunctionID()))
		ctx = logging.NewContextWithLogger(ctx, logger)

		messageAccepted, err := app.MessageReceived(ctx, evt)
		if err != nil {
			logger.Error("message not accepted", "err", err.Error())
			return err
		}

		logger.Info("publishing message", "topic", messageAccepted.TopicName())
		err = messenger.PublishOnTopic(ctx, messageAccepted)
		if err != nil {
			logger.Error("failed to publish message", "err", err.Error())
			return err
		}

		return nil
	}
}

func newTopicMessageHandler(messenger messaging.MsgContext, app application.App) messaging.TopicMessageHandler {
	return func(ctx context.Context, msg amqp.Delivery, logger *slog.Logger) {
		var err error

		ctx, span := tracer.Start(ctx, "receive-message")
		defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

		_, ctx, logger = o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

		logger.Debug("received message", "body", string(msg.Body))

		evt := events.MessageAccepted{}

		err = json.Unmarshal(msg.Body, &evt)
		if err != nil {
			logger.Error("unable to unmarshal incoming message", "err", err.Error())
			return
		}

		err = evt.Error()
		if err != nil {
			logger.Warn("received malformed topic message", "err", err.Error())
			return
		}

		logger = logger.With(slog.String("function_id", evt.Function))
		ctx = logging.NewContextWithLogger(ctx, logger)

		err = app.MessageAccepted(ctx, evt, messenger)
		if err != nil {
			logger.Error("failed to handle message", "err", err.Error())
		}
	}
}

func fatal(ctx context.Context, msg string, err error) {
	logger := logging.GetFromContext(ctx)
	logger.Error(msg, "err", err.Error())
	os.Exit(1)
}
