package digdir

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/liberator/pkg/kubernetes"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	httpRequestTimeout = 30 * time.Second
)

type Client struct {
	HttpClient *http.Client
	Signer     jose.Signer
	Config     *config.Config
	instance   clients.Instance
}

func (c Client) Register(ctx context.Context, payload types.ClientRegistration) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients", c.Config.DigDir.Admin.BaseURL)
	registration := &types.ClientRegistration{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling register client payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPost, endpoint, jsonPayload, registration); err != nil {
		return nil, fmt.Errorf("registering ID-porten client: %w", err)
	}

	return registration, nil
}

func (c Client) ClientExists(desired clients.Instance, ctx context.Context) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients", c.Config.DigDir.Admin.BaseURL)
	clients := make([]types.ClientRegistration, 0)

	if err := c.request(ctx, http.MethodGet, endpoint, nil, &clients); err != nil {
		return nil, fmt.Errorf("fetching list of clients from Digdir: %w", err)
	}

	for _, actual := range clients {
		if clientMatches(actual, desired) {
			desired.GetStatus().SetClientID(actual.ClientID)
			return &actual, nil
		}
	}
	return nil, nil
}

func (c Client) Update(ctx context.Context, payload types.ClientRegistration, clientID string) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients/%s", c.Config.DigDir.Admin.BaseURL, clientID)
	registration := &types.ClientRegistration{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling update client payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPut, endpoint, jsonPayload, registration); err != nil {
		return nil, fmt.Errorf("updating ID-porten client: %w", err)
	}
	return registration, nil
}

func (c Client) Delete(ctx context.Context, clientID string) error {
	endpoint := fmt.Sprintf("%s/clients/%s", c.Config.DigDir.Admin.BaseURL, clientID)
	if err := c.request(ctx, http.MethodDelete, endpoint, nil, nil); err != nil {
		return fmt.Errorf("deleting ID-porten client: %w", err)
	}
	return nil
}

func (c Client) RegisterKeys(ctx context.Context, clientID string, payload *jose.JSONWebKeySet) (*types.JwksResponse, error) {
	endpoint := fmt.Sprintf("%s/clients/%s/jwks", c.Config.DigDir.Admin.BaseURL, clientID)
	response := &types.JwksResponse{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling JWKS payload: %w", err)
	}
	if err := c.request(ctx, http.MethodPost, endpoint, jsonPayload, response); err != nil {
		return nil, fmt.Errorf("registering JWKS for client: %w", err)
	}
	return response, nil
}

func (c Client) GetAccessibleScopes(ctx context.Context) ([]types.Scope, error) {
	endpoint := fmt.Sprintf("%s/scopes/access/all", c.Config.DigDir.Admin.BaseURL)
	scopes := make([]types.Scope, 0)

	if err := c.request(ctx, http.MethodGet, endpoint, nil, &scopes); err != nil {
		return nil, fmt.Errorf("fetching scopes: %w", err)
	}
	return scopes, nil
}

func (c Client) request(ctx context.Context, method string, endpoint string, payload []byte, unmarshalTarget interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, httpRequestTimeout)
	defer cancel()

	token, err := c.getAuthToken(ctx)
	if err != nil {
		return fmt.Errorf("getting token from digdir: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("creating client %s request: %w", method, err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing %s request to %s: %w", method, endpoint, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading server response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server responded with %s: %s", resp.Status, body)
	}

	if unmarshalTarget != nil {
		if err := json.Unmarshal(body, &unmarshalTarget); err != nil {
			return fmt.Errorf("unmarshalling: %w", err)
		}
	}
	return nil
}

func NewClient(httpClient *http.Client, signer jose.Signer, config *config.Config, instance clients.Instance) Client {
	return Client{
		httpClient,
		signer,
		config,
		instance,
	}
}

func clientMatches(actual types.ClientRegistration, desired clients.Instance) bool {
	descriptionMatches := actual.Description == kubernetes.UniformResourceName(desired)
	integrationTypeMatches := actual.IntegrationType == clients.GetIntegrationType(desired)

	return descriptionMatches && integrationTypeMatches
}
