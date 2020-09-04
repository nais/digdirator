package idporten

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/idporten/types"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	httpRequestTimeout = 3 * time.Minute
)

type Client struct {
	Signer jose.Signer
	Config config.Config
}

func (c Client) Register(ctx context.Context, payload types.ClientRegistration) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients", c.Config.DigDir.IDPorten.ApiEndpoint)
	registration := &types.ClientRegistration{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling register client payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPost, endpoint, jsonPayload, &registration); err != nil {
		return nil, fmt.Errorf("updating ID-porten client: %w", err)
	}

	return registration, nil
}

func (c Client) ClientExists(clientID string, ctx context.Context) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients", c.Config.DigDir.IDPorten.ApiEndpoint)
	clients := make([]types.ClientRegistration, 0)

	if err := c.request(ctx, http.MethodGet, endpoint, nil, &clients); err != nil {
		return nil, fmt.Errorf("updating ID-porten client: %w", err)
	}

	for _, client := range clients {
		if client.Description == clientID {
			return &client, nil
		}
	}
	return nil, nil
}

func (c Client) Update(ctx context.Context, payload types.ClientRegistration, clientID string) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients/%s", c.Config.DigDir.IDPorten.ApiEndpoint, clientID)
	registration := &types.ClientRegistration{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling update client payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPut, endpoint, jsonPayload, &registration); err != nil {
		return nil, fmt.Errorf("updating ID-porten client: %w", err)
	}
	return registration, nil
}

func (c Client) Delete(ctx context.Context, clientID string) error {
	endpoint := fmt.Sprintf("%s/clients/%s", c.Config.DigDir.IDPorten.ApiEndpoint, clientID)
	if err := c.request(ctx, http.MethodDelete, endpoint, nil, nil); err != nil {
		return fmt.Errorf("deleting ID-porten client: %w", err)
	}
	return nil
}

func (c Client) RegisterKeys(ctx context.Context, clientID string, payload *jose.JSONWebKeySet) (*types.RegisterJwksResponse, error) {
	endpoint := fmt.Sprintf("%s/clients/%s/jwks", c.Config.DigDir.IDPorten.ApiEndpoint, clientID)
	response := &types.RegisterJwksResponse{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling JWKS payload: %w", err)
	}
	if err := c.request(ctx, http.MethodPost, endpoint, jsonPayload, &response); err != nil {
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

	resp, err := http.DefaultClient.Do(req)
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

func NewClient(signer jose.Signer, config config.Config) Client {
	return Client{
		signer,
		config,
	}
}
