package idporten

import (
	"context"
	"github.com/google/uuid"
	"github.com/nais/digdirator/pkg/crypto"
	"gopkg.in/square/go-jose.v2/jwt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func (c Client) getAuthToken(ctx context.Context) (string, error) {
	claims := jwt.Claims{
		Issuer:    "",
		Subject:   "",
		Audience:  nil,
		Expiry:    nil,
		NotBefore: nil,
		IssuedAt:  nil,
		ID:        uuid.New().String(),
	}

	token, err := crypto.GenerateJwt(c.Signer, claims)
	if err != nil {
		return "", err
	}

	endpoint := c.Config.DigDir.Auth.Endpoint + "/token"

	req, err := authRequest(ctx, endpoint, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// todo - parse token from response, (cache?), return token
	//  - is the client configured to return a by-reference or self-contained token?
	//  - https://difi.github.io/felleslosninger/oidc_auth_server-to-server-oauth2.html#2-send-jwt-til-token-endepunktet
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
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
