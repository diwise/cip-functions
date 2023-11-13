package main

import (
	"context"
	"flag"
	"io"
	"os"

	"github.com/diwise/cip-functions/internal/pkg/application"
	"github.com/diwise/cip-functions/internal/pkg/application/messageprocessor"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
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
		return nil, nil, err
	}

	app := application.New(msgproc, functionsRegistry)

	needToDecideThis := "application/json"
	msgctx.RegisterCommandHandler(needToDecideThis, newCommandHandler(msgctx, app))

	routingKey := "message.accepted"
	msgctx.RegisterTopicMessageHandler(routingKey, newTopicMessageHandler(msgctx, app))

	return app, nil
}

func fatal(ctx context.Context, msg string, err error) {
	logger := logging.GetFromContext(ctx)
	logger.Error(msg, "err", err.Error())
	os.Exit(1)
}
