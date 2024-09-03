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
	FindByID(ctx context.Context, id, thingType string) (Thing, error)
	FindRelatedThings(ctx context.Context, id, thingType string) ([]Thing, error)
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

func (tc ClientImpl) FindByID(ctx context.Context, id, thingType string) (Thing, error) {
	jar, err := tc.findByID(ctx, id, thingType)
	if err != nil {
		return Thing{}, err
	}

	t := Thing{}

	err = json.Unmarshal(jar.Data, &t)
	if err != nil {
		return t, err
	}

	return t, nil
}

func (tc ClientImpl) FindRelatedThings(ctx context.Context, id, thingType string) ([]Thing, error) {
	jar, err := tc.findByID(ctx, id, thingType)
	if err != nil {
		return nil, err
	}
	return jar.Included, nil
}

func (tc ClientImpl) findByID(ctx context.Context, id, thingType string) (*JsonApiResponse, error) {
	var err error
	ctx, span := tracer.Start(ctx, "find-thing-by-id")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	log := logging.GetFromContext(ctx)

	url := fmt.Sprintf("%s/%s/urn:diwise:%s:%s", tc.url, "api/v0/things", strings.ToLower(thingType), strings.ToLower(id))

	cachedItem, found := tc.cache.Get(url)
	if found {
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

	log.Debug(fmt.Sprintf("response body: %s", string(body)))

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
	ThingID    string         `json:"thing_id"`
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Location   Location       `json:"location"`
	Tenant     string         `json:"tenant"`
	Properties map[string]any `json:"properties,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (t *Thing) UnmarshalJSON(data []byte) error {
	props := make(map[string]any)
	err := json.Unmarshal(data, &props)
	if err != nil {
		return nil
	}

	if thingID, ok := props["thing_id"]; ok {
		t.ThingID = thingID.(string)
		delete(props, "thing_id")
	}

	if id, ok := props["id"]; ok {
		t.ID = id.(string)
		delete(props, "id")
	}

	if type_, ok := props["type"]; ok {
		t.Type = type_.(string)
		delete(props, "type")
	}

	if loc, ok := props["location"]; ok {
		b, err := json.Marshal(loc)
		if err != nil {
			return err
		}
		var loc Location
		err = json.Unmarshal(b, &loc)
		if err != nil {
			return err
		}
		t.Location = loc
		delete(props, "location")
	}

	if tenant, ok := props["tenant"]; ok {
		t.Tenant = tenant.(string)
		delete(props, "tenant")
	}

	if len(props) > 0 {
		if len(t.Properties) == 0 {
			t.Properties = make(map[string]any)
		}
		for k, v := range props {
			if k == "properties" {
				if properties, ok := v.(map[string]any); ok {
					for pk, pv := range properties {
						t.Properties[pk] = pv
					}
				}
			} else {
				t.Properties[k] = v
			}
		}
	}

	return nil
}
