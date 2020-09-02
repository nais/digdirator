package idporten

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/nais/digdirator/pkg/crypto"
	"gopkg.in/square/go-jose.v2/jwt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (c Client) getAuthToken(ctx context.Context) (*TokenResponse, error) {
	claims := jwt.Claims{
		Issuer:    "oidc_nav_portal_integrasjons_admin",
		Audience:  []string{"https://oidc-ver2.difi.no/idporten-oidc-provider/"},
		Expiry:    jwt.NewNumericDate(time.Now().Add(2 * time.Minute)),
		NotBefore: jwt.NewNumericDate(time.Now()),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ID:        uuid.New().String(),
	}

	token, err := crypto.GenerateJwt(c.Signer, claims)
	if err != nil {
		return nil, err
	}

	endpoint := c.Config.DigDir.Auth.Endpoint + "/token"

	req, err := authRequest(ctx, endpoint, token)
	if err != nil {
		return nil, fmt.Errorf("creating auth request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	return &tokenResponse, nil
}

func authRequest(ctx context.Context, endpoint, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, authQueryParams(token))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func authQueryParams(token string) io.Reader {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	data.Set("assertion", token)
	return strings.NewReader(data.Encode())
}
