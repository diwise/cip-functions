package things

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/oauth2/clientcredentials"
)

var tracer = otel.Tracer("things-client")

var ErrThingNotFound = fmt.Errorf("thing not found")

type ClientImpl struct {
	url               string
	clientCredentials *clientcredentials.Config
	httpClient        http.Client
	cache             *Cache
}

//go:generate moq -rm -out client_mock.go . Client
type Client interface {
	FindByID(ctx context.Context, thingID string) (Thing, error)
	FindRelatedThings(ctx context.Context, thingID string) ([]Thing, error)
}

func NewClient(ctx context.Context, url, oauthTokenURL, oauthClientID, oauthClientSecret string) (*ClientImpl, error) {
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

	c := NewCache()
	c.Cleanup(5 * time.Minute)

	return &ClientImpl{
		url:               strings.TrimSuffix(url, "/"),
		clientCredentials: oauthConfig,
		httpClient: http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		cache: c,
	}, nil
}

func (tc ClientImpl) FindByID(ctx context.Context, thingID string) (Thing, error) {
	t := Thing{}

	jar, err := tc.findByID(ctx, thingID)
	if err != nil {
		return t, err
	}

	err = json.Unmarshal(jar.Data, &t)
	if err != nil {
		return t, err
	}

	return t, nil
}

func (tc ClientImpl) FindRelatedThings(ctx context.Context, thingID string) ([]Thing, error) {
	jar, err := tc.findByID(ctx, thingID)
	if err != nil {
		return nil, err
	}
	return jar.Included, nil
}

func (tc ClientImpl) findByID(ctx context.Context, thingID string) (*JsonApiResponse, error) {
	var err error
	ctx, span := tracer.Start(ctx, "find-thing-by-id")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	log := logging.GetFromContext(ctx)

	url := fmt.Sprintf("%s/%s/%s", tc.url, "api/v0/things", thingID)

	cachedItem, found := tc.cache.Get(url)
	if found {
		log.Debug(fmt.Sprintf("found response for %s in cache", url))

		jar, ok := cachedItem.(JsonApiResponse)
		if ok {
			return &jar, nil
		}

		log.Warn(fmt.Sprintf("found response for %s in cache but could not cast to JsonApiResponse", url))
	}

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
		err = fmt.Errorf("failed to retrieve thing: %w", err)
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

	tc.cache.Set(url, jar, 1*time.Minute)

	return &jar, nil
}

type JsonApiResponse struct {
	Data     json.RawMessage `json:"data"`
	Included []Thing         `json:"included,omitempty"`
}

type Thing struct {
	Id       string   `json:"id"`
	Type     string   `json:"type"`
	Location Location `json:"location"`
	Tenant   string   `json:"tenant,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
