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

	cache "github.com/Code-Hex/go-generics-cache"
	"github.com/go-jose/go-jose/v4"
	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/kubernetes"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/retry"
)

const (
	httpRequestTimeout = 30 * time.Second
	retryInitialDelay  = 1 * time.Second
	retryMaxAttempts   = 5
)

var (
	ErrServer = errors.New("ServerError")
	ErrClient = errors.New("ClientError")

	scopeCache = cache.New[string, bool]()
)

type Error struct {
	Err        error
	Status     string
	StatusCode int
	Message    string
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
}

func NewClient(config *config.Config, httpClient *http.Client, signer jose.Signer) (Client, error) {
	return Client{
		Config:     config,
		HttpClient: httpClient,
		Signer:     signer,
	}, nil
}

func (c Client) Register(ctx context.Context, payload types.ClientRegistration) (*types.ClientRegistration, error) {
	endpoint := c.endpoint("clients")
	registration := &types.ClientRegistration{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPost, endpoint, jsonPayload, registration); err != nil {
		return nil, err
	}

	return registration, nil
}

func (c Client) GetRegistration(desired clients.Instance, ctx context.Context, clusterName string) (*types.ClientRegistration, error) {
	endpoint := c.endpoint("clients")
	clientRegistrations := make([]types.ClientRegistration, 0)

	if err := c.request(ctx, http.MethodGet, endpoint, nil, &clientRegistrations); err != nil {
		return nil, err
	}

	for _, actual := range clientRegistrations {
		if clientMatches(actual, desired, clusterName) {
			desired.GetStatus().ClientID = actual.ClientID
			return &actual, nil
		}
	}
	return nil, nil
}

func (c Client) Exists(ctx context.Context, desired clients.Instance, clusterName string) (bool, error) {
	registration, err := c.GetRegistration(desired, ctx, clusterName)
	if err != nil {
		return false, err
	}

	return registration != nil, nil
}

func (c Client) Update(ctx context.Context, payload types.ClientRegistration, clientID string) (*types.ClientRegistration, error) {
	endpoint := c.endpoint("clients", clientID)
	registration := &types.ClientRegistration{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPut, endpoint, jsonPayload, registration); err != nil {
		return nil, err
	}
	return registration, nil
}

func (c Client) Delete(ctx context.Context, clientID string) error {
	endpoint := c.endpoint("clients", clientID)
	if err := c.request(ctx, http.MethodDelete, endpoint, nil, nil); err != nil {
		return err
	}
	return nil
}

func (c Client) GetKeys(ctx context.Context, clientID string) (*types.JwksResponse, error) {
	endpoint := c.endpoint("clients", clientID, "jwks")
	response := &types.JwksResponse{}

	if err := c.request(ctx, http.MethodGet, endpoint, nil, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c Client) RegisterKeys(ctx context.Context, clientID string, payload *jose.JSONWebKeySet) (*types.JwksResponse, error) {
	endpoint := c.endpoint("clients", clientID, "jwks")
	response := &types.JwksResponse{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	if err := c.request(ctx, http.MethodPost, endpoint, jsonPayload, response); err != nil {
		return nil, err
	}
	return response, nil
}

// CanAccessScope checks if the authenticated organization can access given scope.
func (c Client) CanAccessScope(ctx context.Context, scope nais_io_v1.ConsumedScope) (bool, error) {
	if access, ok := scopeCache.Get(scope.Name); ok {
		return access, nil
	}

	// cache miss, fetch fresh scope data from DigDir
	_, err := c.GetAccessibleScopes(ctx)
	if err != nil {
		return false, fmt.Errorf("get accessible scopes: %w", err)
	}
	_, err = c.GetOpenScopes(ctx)
	if err != nil {
		return false, fmt.Errorf("get open scopes: %w", err)
	}

	if access, ok := scopeCache.Get(scope.Name); ok {
		return access, nil
	}

	return false, nil
}

// GetAccessibleScopes returns all scopes that the authenticated organization has been granted access to.
func (c Client) GetAccessibleScopes(ctx context.Context) ([]types.Scope, error) {
	endpoint := c.endpoint("scopes", "access", "all")

	s := make([]types.Scope, 0)
	if err := c.request(ctx, http.MethodGet, endpoint, nil, &s); err != nil {
		return nil, err
	}

	for _, scope := range s {
		// cache with reasonable expiration time to prevent stale data
		scopeCache.Set(scope.Scope, scope.IsAccessible(), cache.WithExpiration(10*time.Minute))
	}

	return s, nil
}

// GetOpenScopes returns all scopes that are accessible to any organization.
func (c Client) GetOpenScopes(ctx context.Context) ([]types.ScopeRegistration, error) {
	endpoint := c.endpoint("scopes", "all") + "?accessible_for_all=true"

	s := make([]types.ScopeRegistration, 0)
	if err := c.request(ctx, http.MethodGet, endpoint, nil, &s); err != nil {
		return nil, err
	}

	for _, scope := range s {
		scopeCache.Set(scope.Name, scope.AccessibleForAll)
	}

	return s, nil
}

func (c Client) GetScopes(ctx context.Context) ([]types.ScopeRegistration, error) {
	endpoint := c.endpoint("scopes") + "?inactive=true"
	scopes := make([]types.ScopeRegistration, 0)

	if err := c.request(ctx, http.MethodGet, endpoint, nil, &scopes); err != nil {
		return nil, err
	}

	return scopes, nil
}

func (c Client) RegisterScope(ctx context.Context, payload types.ScopeRegistration) (*types.ScopeRegistration, error) {
	endpoint := c.endpoint("scopes")
	registration := &types.ScopeRegistration{}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPost, endpoint, jsonPayload, registration); err != nil {
		return nil, err
	}
	return registration, nil
}

func (c Client) UpdateScope(ctx context.Context, payload types.ScopeRegistration, scope string) (*types.ScopeRegistration, error) {
	endpoint := c.endpoint("scopes") + "?scope=" + url.QueryEscape(scope)
	registration := &types.ScopeRegistration{}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	if err := c.request(ctx, http.MethodPut, endpoint, jsonPayload, registration); err != nil {
		return nil, err
	}

	return registration, nil
}

func (c Client) DeleteScope(ctx context.Context, scope string) (*types.ScopeRegistration, error) {
	endpoint := c.endpoint("scopes") + "?scope=" + url.QueryEscape(scope)
	actualScopesRegistration := &types.ScopeRegistration{}

	if err := c.request(ctx, http.MethodDelete, endpoint, nil, &actualScopesRegistration); err != nil {
		return nil, err
	}
	return actualScopesRegistration, nil
}

func (c Client) GetScopeACL(ctx context.Context, scope string) (*[]types.ConsumerRegistration, error) {
	endpoint := c.endpoint("scopes", "access") + "?scope=" + url.QueryEscape(scope)
	registration := &[]types.ConsumerRegistration{}
	if err := c.request(ctx, http.MethodGet, endpoint, nil, registration); err != nil {
		return nil, err
	}
	return registration, nil
}

func (c Client) AddToScopeACL(ctx context.Context, scope, consumerOrgno string) (*types.ConsumerRegistration, error) {
	endpoint := c.endpoint("scopes", "access", consumerOrgno) + "?scope=" + url.QueryEscape(scope)
	registration := &types.ConsumerRegistration{}

	if err := c.request(ctx, http.MethodPut, endpoint, nil, registration); err != nil {
		return nil, err
	}
	return registration, nil
}

func (c Client) DeactivateConsumer(ctx context.Context, scope, consumerOrgno string) (*types.ConsumerRegistration, error) {
	endpoint := c.endpoint("scopes", "access", consumerOrgno) + "?scope=" + url.QueryEscape(scope)
	registration := &types.ConsumerRegistration{}

	if err := c.request(ctx, http.MethodDelete, endpoint, []byte{}, registration); err != nil {
		return nil, err
	}
	return registration, nil
}

func (c Client) endpoint(path ...string) string {
	return c.Config.DigDir.Admin.ApiV1URL().JoinPath(path...).String()
}

func (c Client) request(ctx context.Context, method string, endpoint string, payload []byte, unmarshalTarget any) error {
	retryable := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, httpRequestTimeout)
		defer cancel()

		token, err := c.getAuthToken(ctx)
		if err != nil {
			return fmt.Errorf("get auth token: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("creating %s request: %w", method, err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
		req.Header.Add("Content-Type", "application/json")

		resp, err := c.HttpClient.Do(req)
		if err != nil {
			return fmt.Errorf("doing %s request to %s: %w", method, endpoint, err)
		}

		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			err = &Error{
				Err:        ErrClient,
				Message:    string(body),
				Status:     resp.Status,
				StatusCode: resp.StatusCode,
			}
		} else if resp.StatusCode >= 500 {
			err = &Error{
				Err:        ErrServer,
				Message:    string(body),
				Status:     resp.Status,
				StatusCode: resp.StatusCode,
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

func clientMatches(actual types.ClientRegistration, desired clients.Instance, clusterName string) bool {
	if desired.GetStatus() != nil && desired.GetStatus().ClientID != "" {
		return actual.ClientID == desired.GetStatus().ClientID
	}

	// We don't have an existing client ID, so we'll have to do best-effort matching.
	descriptionMatches := actual.Description == kubernetes.UniformResourceName(desired, clusterName)
	integrationTypeMatches := actual.IntegrationType == clients.GetIntegrationType(desired)
	return descriptionMatches && integrationTypeMatches
}
