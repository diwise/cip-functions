package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/oauth2/clientcredentials"
)

var tracer = otel.Tracer("things-client")

var ErrThingNotFound = fmt.Errorf("thing not found")

type ThingsClient struct {
	url               string
	clientCredentials *clientcredentials.Config
	httpClient        http.Client
}

func NewThingsClient(ctx context.Context, url, oauthTokenURL, oauthClientID, oauthClientSecret string) (*ThingsClient, error) {
	oauthConfig := &clientcredentials.Config{
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSecret,
		TokenURL:     oauthTokenURL,
	}

	token, err := oauthConfig.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client credentials from %s: %w", oauthConfig.TokenURL, err)
	}

	if !token.Valid() {
		return nil, fmt.Errorf("an invalid token was returned from %s", oauthTokenURL)
	}

	return &ThingsClient{
		url:               url,
		clientCredentials: oauthConfig,
		httpClient: http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}, nil
}

func (tc ThingsClient) FindRelatedThings(ctx context.Context, thingID string) ([]Thing, error) {
	jar, err := tc.findByID(ctx, thingID)
	if err != nil {
		return nil, err
	}
	return jar.Included, nil
}

func (tc ThingsClient) findByID(ctx context.Context, thingID string) (*JsonApiResponse, error) {
	var err error
	ctx, span := tracer.Start(ctx, "find-thing-by-id")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	url := tc.url + "/api/v0/things/%s" + thingID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		err = fmt.Errorf("failed to create http request: %w", err)
		return nil, err
	}

	req.Header.Add("Accept", "application/vnd.api+json")

	if tc.clientCredentials != nil {
		token, err := tc.clientCredentials.Token(ctx)
		if err != nil {
			err = fmt.Errorf("failed to get client credentials from %s: %w", tc.clientCredentials.TokenURL, err)
			return nil, err
		}

		req.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
	}

	resp, err := tc.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("failed to retrieve device information from devEUI: %w", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		err = fmt.Errorf("request failed, not authorized")
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrThingNotFound
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("request failed with status code %d", resp.StatusCode)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response body: %w", err)
		return nil, err
	}

	jar := JsonApiResponse{}
	err = json.Unmarshal(body, &jar)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal response body: %w", err)
		return nil, err
	}

	return &jar, nil
}

type JsonApiResponse struct {
	Data     json.RawMessage `json:"data"`
	Included []Thing         `json:"included,omitempty"`
}

type Thing struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}