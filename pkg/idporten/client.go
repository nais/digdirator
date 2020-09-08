package idporten

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/idporten/types"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	httpRequestTimeout = 3 * time.Minute // todo - change
)

type Client struct {
	HttpClient *http.Client
	Signer     jose.Signer
	Config     *config.Config
}

func (c Client) Register(ctx context.Context, payload types.ClientRegistration) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients", c.Config.DigDir.IDPorten.BaseURL)
	registration := &types.ClientRegistration{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling register client payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPost, endpoint, jsonPayload, registration); err != nil {
		return nil, fmt.Errorf("updating ID-porten client: %w", err)
	}

	return registration, nil
}

func (c Client) ClientExists(desired *v1.IDPortenClient, ctx context.Context) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients", c.Config.DigDir.IDPorten.BaseURL)
	clients := make([]types.ClientRegistration, 0)

	if err := c.request(ctx, http.MethodGet, endpoint, nil, &clients); err != nil {
		return nil, fmt.Errorf("updating ID-porten client: %w", err)
	}

	for _, actual := range clients {
		if clientMatches(actual, desired) {
			return &actual, nil
		}
	}
	return nil, nil
}

func (c Client) Update(ctx context.Context, payload types.ClientRegistration, clientID string) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients/%s", c.Config.DigDir.IDPorten.BaseURL, clientID)
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
	endpoint := fmt.Sprintf("%s/clients/%s", c.Config.DigDir.IDPorten.BaseURL, clientID)
	if err := c.request(ctx, http.MethodDelete, endpoint, nil, nil); err != nil {
		return fmt.Errorf("deleting ID-porten client: %w", err)
	}
	return nil
}

func (c Client) RegisterKeys(ctx context.Context, clientID string, payload *jose.JSONWebKeySet) (*types.JwksResponse, error) {
	endpoint := fmt.Sprintf("%s/clients/%s/jwks", c.Config.DigDir.IDPorten.BaseURL, clientID)
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

func NewClient(httpClient *http.Client, signer jose.Signer, config *config.Config) Client {
	return Client{
		httpClient,
		signer,
		config,
	}
}

func clientMatches(actual types.ClientRegistration, desired *v1.IDPortenClient) bool {
	idExists := len(desired.Status.ClientID) > 0
	idMatches := actual.ClientID == desired.Status.ClientID
	descriptionMatches := actual.Description == desired.ClientDescription()

	return (idExists && idMatches) || descriptionMatches
}
