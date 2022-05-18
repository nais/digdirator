package digdir

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/nais/digdirator/pkg/crypto"
)

const (
	grantType                 = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	applicationFormUrlEncoded = "application/x-www-form-urlencoded"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

type customClaims struct {
	jwt.Claims
	Scope string `json:"scope"`
}

func (c Client) getAuthToken(ctx context.Context) (*TokenResponse, error) {
	token, err := crypto.GenerateJwt(c.Signer, c.claims())
	if err != nil {
		return nil, fmt.Errorf("generating JWT for ID-porten auth: %w", err)
	}

	endpoint := c.Config.DigDir.IDPorten.Metadata.TokenEndpoint

	req, err := authRequest(ctx, endpoint, token)
	if err != nil {
		return nil, fmt.Errorf("creating auth request: %w", err)
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing http request to ID-porten: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading server response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server responded with %s: %s", resp.Status, body)
	}

	tokenResponse := &TokenResponse{}
	if err := json.Unmarshal(body, tokenResponse); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	return tokenResponse, nil
}

func (c Client) claims() customClaims {
	var clientID string
	var scopes string

	switch c.instance.(type) {
	case *nais_io_v1.IDPortenClient:
		clientID = c.Config.DigDir.IDPorten.ClientID
		scopes = c.Config.DigDir.IDPorten.Scopes
	case *nais_io_v1.MaskinportenClient:
		clientID = c.Config.DigDir.Maskinporten.ClientID
		scopes = c.Config.DigDir.Maskinporten.Scopes
	}

	return customClaims{
		Claims: jwt.Claims{
			Issuer:    clientID,
			Audience:  []string{c.Config.DigDir.IDPorten.Metadata.Issuer},
			Expiry:    jwt.NewNumericDate(time.Now().Add(2 * time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
		Scope: scopes,
	}
}

func authRequest(ctx context.Context, endpoint, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, authQueryParams(token))
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}
	req.Header.Set("Content-Type", applicationFormUrlEncoded)
	return req, nil
}

func authQueryParams(token string) io.Reader {
	data := url.Values{
		"grant_type": []string{grantType},
		"assertion":  []string{token},
	}
	return strings.NewReader(data.Encode())
}
