package idporten

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/digdirator/pkg/config"
	"gopkg.in/square/go-jose.v2"
	"net/http"
	"time"
)

type Client struct {
	Signer jose.Signer
	Config config.Config
}

func (c Client) Register(ctx context.Context, payload RegisterClientRequest) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	token, err := c.getAuthToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get token from digdir: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// todo - POST to /clients endpoint
	endpoint := c.Config.DigDir.IDPorten.Endpoint + "/clients"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create post request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register idporten client: %w", err)
	}
	defer resp.Body.Close()

	// todo - handle status codes

	return nil
}

func (c Client) List(ctx context.Context) error {
	panic("implement me")
}

func (c Client) Update(ctx context.Context, payload RegisterClientRequest) error {
	panic("implement me")
}

func (c Client) Delete(ctx context.Context) error {
	panic("implement me")
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
