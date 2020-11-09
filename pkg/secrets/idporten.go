package secrets

import (
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/config"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"
)

const (
	IDPortenJwkKey       = "IDPORTEN_CLIENT_JWK"
	IDPortenClientID     = "IDPORTEN_CLIENT_ID"
	IDPortenWellKnownURL = "IDPORTEN_WELL_KNOWN_URL"
	IDPortenRedirectURI  = "IDPORTEN_REDIRECT_URI"
)

func IDPortenStringData(jwk jose.JSONWebKey, instance *v1.IDPortenClient) (map[string]string, error) {
	wellKnownURL := viper.GetString(config.DigDirAuthBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		IDPortenJwkKey:       string(jwkJson),
		IDPortenWellKnownURL: wellKnownURL,
		IDPortenClientID:     instance.StatusClientID(),
		IDPortenRedirectURI:  instance.Spec.RedirectURI,
	}, nil
}
