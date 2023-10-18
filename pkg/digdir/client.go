package digdir

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"
	"gopkg.in/square/go-jose.v2"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir/scopes"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/retry"
)

const (
	httpRequestTimeout = 30 * time.Second
	retryInitialDelay  = 1 * time.Second
	retryMaxAttempts   = 5
)

var (
	ServerError = errors.New("ServerError")
	ClientError = errors.New("ClientError")
)

type Error struct {
	Err     error
	Status  string
	Message string
}

func (in *Error) Error() string {
	return fmt.Sprintf("HTTP %s: %s", in.Status, in.Message)
}

func (in *Error) Unwrap() error {
	return in.Err
}

type Client struct {
	HttpClient *http.Client
	Signer     jose.Signer
	Config     *config.Config
	instance   clients.Instance
	ClientId   []byte
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

func (c Client) GetRegistration(desired clients.Instance, ctx context.Context, clusterName string) (*types.ClientRegistration, error) {
	endpoint := fmt.Sprintf("%s/clients", c.Config.DigDir.Admin.BaseURL)
	clientRegistrations := make([]types.ClientRegistration, 0)

	if err := c.request(ctx, http.MethodGet, endpoint, nil, &clientRegistrations); err != nil {
		return nil, fmt.Errorf("fetching list of clients from Digdir: %w", err)
	}

	for _, actual := range clientRegistrations {
		if clientMatches(actual, desired, clusterName) {
			desired.GetStatus().SetClientID(actual.ClientID)
			return &actual, nil
		}
	}
	return nil, nil
}

func (c Client) Exists(ctx context.Context, desired clients.Instance, clusterName string) (bool, error) {
	registration, err := c.GetRegistration(desired, ctx, clusterName)
	if err != nil {
		return false, fmt.Errorf("getting client registration: %w", err)
	}

	return registration != nil, nil
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

func (c Client) GetKeys(ctx context.Context, clientID string) (*types.JwksResponse, error) {
	endpoint := fmt.Sprintf("%s/clients/%s/jwks", c.Config.DigDir.Admin.BaseURL, clientID)
	response := &types.JwksResponse{}

	if err := c.request(ctx, http.MethodGet, endpoint, nil, response); err != nil {
		return nil, fmt.Errorf("getting JWKS for client: %w", err)
	}
	return response, nil
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
	s := make([]types.Scope, 0)

	if err := c.request(ctx, http.MethodGet, endpoint, nil, &s); err != nil {
		return nil, fmt.Errorf("fetching scopes: %w", err)
	}
	return s, nil
}

func (c Client) request(ctx context.Context, method string, endpoint string, payload []byte, unmarshalTarget interface{}) error {
	retryable := func(ctx context.Context) error {
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

		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading server response: %w", err)
		}

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			err = &Error{
				Err:     ClientError,
				Message: string(body),
				Status:  resp.Status,
			}
		} else if resp.StatusCode >= 500 {
			err = &Error{
				Err:     ServerError,
				Message: string(body),
				Status:  resp.Status,
			}
		}
		if err != nil {
			return retry.RetryableError(err)
		}

		if unmarshalTarget != nil {
			if err := json.Unmarshal(body, &unmarshalTarget); err != nil {
				return fmt.Errorf("unmarshalling: %w", err)
			}
		}
		return nil
	}

	return retry.Fibonacci(retryInitialDelay).
		WithMaxAttempts(retryMaxAttempts).
		Do(ctx, retryable)
}

func NewClient(httpClient *http.Client, signer jose.Signer, config *config.Config, instance clients.Instance, clientId []byte) Client {
	return Client{
		httpClient,
		signer,
		config,
		instance,
		clientId,
	}
}

func clientMatches(actual types.ClientRegistration, desired clients.Instance, clusterName string) bool {
	if desired.GetStatus() != nil && desired.GetStatus().GetClientID() != "" {
		return actual.ClientID == desired.GetStatus().GetClientID()
	}

	// We don't have an existing client ID, so we'll have to do best-effort matching.
	descriptionMatches := actual.Description == kubernetes.UniformResourceName(desired, clusterName)
	integrationTypeMatches := actual.IntegrationType == clients.GetIntegrationType(desired)
	return descriptionMatches && integrationTypeMatches
}

func (c Client) GetFilteredScopes(desired clients.Instance, ctx context.Context, exposedScopes map[string]nais_io_v1.ExposedScope, clusterName string) (*scopes.ScopeStash, error) {
	endpoint := fmt.Sprintf("%s/scopes?inactive=true", c.Config.DigDir.Admin.BaseURL)
	actualScopesRegistrations := make([]types.ScopeRegistration, 0)
	container := scopes.ScopeStash{}

	if err := c.request(ctx, http.MethodGet, endpoint, nil, &actualScopesRegistrations); err != nil {
		return nil, fmt.Errorf("fetching list of scopes from Digdir: %w", err)
	}
	return container.FilterScopes(actualScopesRegistrations, desired, exposedScopes, clusterName), nil
}

func (c Client) RegisterScope(ctx context.Context, payload types.ScopeRegistration) (*types.ScopeRegistration, error) {
	endpoint := fmt.Sprintf("%s/scopes", c.Config.DigDir.Admin.BaseURL)
	registration := &types.ScopeRegistration{}
	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		return nil, fmt.Errorf("marshalling register scope payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPost, endpoint, jsonPayload, registration); err != nil {
		return nil, fmt.Errorf("registering scope: %w", err)
	}
	return registration, nil
}

func (c Client) UpdateScope(ctx context.Context, payload types.ScopeRegistration, scope string) (*types.ScopeRegistration, error) {
	endpoint := fmt.Sprintf("%s/scopes?scope=%s", c.Config.DigDir.Admin.BaseURL, url.QueryEscape(scope))
	registration := &types.ScopeRegistration{}
	jsonPayload, err := json.Marshal(payload)

	if err != nil {
		return nil, fmt.Errorf("marshalling update scope payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPut, endpoint, jsonPayload, registration); err != nil {
		return nil, fmt.Errorf("updating scope: %w", err)
	}
	return registration, nil
}

func (c Client) DeleteScope(ctx context.Context, scope string) (*types.ScopeRegistration, error) {
	endpoint := fmt.Sprintf("%s/scopes?scope=%s", c.Config.DigDir.Admin.BaseURL, url.QueryEscape(scope))
	actualScopesRegistration := &types.ScopeRegistration{}

	if err := c.request(ctx, http.MethodDelete, endpoint, nil, &actualScopesRegistration); err != nil {
		return nil, fmt.Errorf("deleting scope from Digdir: %w", err)
	}
	return actualScopesRegistration, nil
}

func (c Client) ActivateScope(ctx context.Context, payload types.ScopeRegistration, scope string) (*types.ScopeRegistration, error) {
	endpoint := fmt.Sprintf("%s/scopes?scope=%s", c.Config.DigDir.Admin.BaseURL, url.QueryEscape(scope))
	actualScopesRegistration := &types.ScopeRegistration{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling activate scope payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPut, endpoint, jsonPayload, actualScopesRegistration); err != nil {
		return nil, fmt.Errorf("activate scope: %w", err)
	}
	return actualScopesRegistration, nil

}

func (c Client) GetScopeACL(ctx context.Context, scope string) (*[]types.ConsumerRegistration, error) {
	endpoint := fmt.Sprintf("%s/scopes/access?scope=%s", c.Config.DigDir.Admin.BaseURL, url.QueryEscape(scope))
	registration := &[]types.ConsumerRegistration{}
	if err := c.request(ctx, http.MethodGet, endpoint, nil, registration); err != nil {
		return nil, fmt.Errorf("getting scope access: %w", err)
	}
	return registration, nil
}

func (c Client) AddToScopeACL(ctx context.Context, scope, consumerOrgno string) (*types.ConsumerRegistration, error) {
	endpoint := fmt.Sprintf("%s/scopes/access/%s?scope=%s", c.Config.DigDir.Admin.BaseURL, consumerOrgno, url.QueryEscape(scope))
	registration := &types.ConsumerRegistration{}

	if err := c.request(ctx, http.MethodPut, endpoint, nil, registration); err != nil {
		return nil, fmt.Errorf("updating scope acl: %w", err)
	}
	return registration, nil
}

func (c Client) DeactivateConsumer(ctx context.Context, scope, consumerOrgno string) (*types.ConsumerRegistration, error) {
	endpoint := fmt.Sprintf("%s/scopes/access/%s?scope=%s", c.Config.DigDir.Admin.BaseURL, consumerOrgno, url.QueryEscape(scope))
	registration := &types.ConsumerRegistration{}

	if err := c.request(ctx, http.MethodDelete, endpoint, []byte{}, registration); err != nil {
		return nil, fmt.Errorf("delete consumer from scope: %w", err)
	}
	return registration, nil
}
