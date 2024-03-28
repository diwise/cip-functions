package main

import (
	"context"
	"net/http"
	"os"

	"github.com/diwise/cip-functions/internal/pkg/application"
	"github.com/diwise/cip-functions/internal/pkg/application/things"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage/database"
	"github.com/diwise/cip-functions/internal/pkg/presentation/api"

	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const serviceName string = "cip-functions"

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, _, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	var err error

	msgCtx := createMessagingContextOrDie(ctx)
	defer msgCtx.Close()

	storage := createDatabaseConnectionOrDie(ctx)
	thingsClient := createThingsClientOrDie(ctx)

	_, err = initialize(ctx, msgCtx, thingsClient, storage)
	if err != nil {
		fatal(ctx, "initialization failed", err)
	}

	servicePort := env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080")
	err = http.ListenAndServe(":"+servicePort, api.New())
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
		fatal(ctx, "failed to create things client", err)
	}

	return c
}

func initialize(ctx context.Context, msgctx messaging.MsgContext, tc things.Client, storage storage.Storage) (application.App, error) {
	app := application.New(msgctx, tc, storage)
	err := application.RegisterMessageHandlers(app)
	if err != nil {
		fatal(ctx, "failed to register handlers", err)
	}

	return app, err
}

func fatal(ctx context.Context, msg string, err error) {
	logger := logging.GetFromContext(ctx)
	logger.Error(msg, "err", err.Error())
	os.Exit(1)
}
