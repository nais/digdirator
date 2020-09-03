package idporten

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/digdirator/pkg/config"
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

func (c Client) Register(ctx context.Context, payload ClientRegistration) (*ClientRegistration, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	token, err := c.getAuthToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token from digdir: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, httpRequestTimeout)
	defer cancel()

	endpoint := c.Config.DigDir.IDPorten.ApiEndpoint + "/clients"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create post request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to register idporten client: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading server response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server responded with %s: %s", resp.Status, body)
	}

	registration := &ClientRegistration{}
	if err := json.Unmarshal(body, registration); err != nil {
		return nil, fmt.Errorf("decoding response when creating client: %w", err)
	}

	return registration, nil
}

func (c Client) ClientExists(clientID string, ctx context.Context) (*ClientRegistration, error) {
	ctx, cancel := context.WithTimeout(ctx, httpRequestTimeout)
	defer cancel()

	token, err := c.getAuthToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token from digdir: %w", err)
	}

	endpoint := c.Config.DigDir.IDPorten.ApiEndpoint + "/clients"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating client list/GET request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check existence idporten client: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading server response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server responded with %s: %s", resp.Status, body)
	}

	clients := make([]ClientRegistration, 0)
	if err := json.Unmarshal(body, &clients); err != nil {
		return nil, fmt.Errorf("decoding list of clientregistrations: %w", err)
	}

	for _, client := range clients {
		if client.Description == clientID {
			return &client, nil
		}
	}

	return nil, nil
}

func (c Client) Update(ctx context.Context, payload ClientRegistration) (*ClientRegistration, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	token, err := c.getAuthToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token from digdir: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, httpRequestTimeout)
	defer cancel()

	endpoint := c.Config.DigDir.IDPorten.ApiEndpoint + "/clients"
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create patch request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update idporten client: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading server response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server responded with %s: %s", resp.Status, body)
	}
	registration := &ClientRegistration{}
	if err := json.Unmarshal(body, registration); err != nil {
		return nil, fmt.Errorf("decoding response when creating client: %w", err)
	}

	return registration, nil
}

func (c Client) Delete(ctx context.Context, clientID string) error {
	ctx, cancel := context.WithTimeout(ctx, httpRequestTimeout)
	defer cancel()

	token, err := c.getAuthToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token from digdir: %w", err)
	}

	endpoint := c.Config.DigDir.IDPorten.ApiEndpoint + "/clients" + "/" + clientID
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating client DELETE request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete idporten client: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading server response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server responded with %s: %s", resp.Status, body)
	}

	return nil
}

func (c Client) RegisterKeys(ctx context.Context, payload jose.JSONWebKeySet) (RegisterJwksResponse, error) {
	panic("implement me")
}

func NewClient(signer jose.Signer, config config.Config) Client {
	return Client{
		signer,
		config,
	}
}
