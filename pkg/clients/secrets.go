package clients

import (
	"fmt"
	"strings"

	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/secrets"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"
)

func SecretData(instance Instance, jwk jose.JSONWebKey) (map[string]string, error) {
	var stringData map[string]string
	var err error

	switch v := instance.(type) {
	case *nais_io_v1.IDPortenClient:
		stringData, err = idPortenClientSecretData(v, jwk)
	case *nais_io_v1.MaskinportenClient:
		stringData, err = maskinportenClientSecretData(v, jwk)
	}

	if err != nil {
		return nil, err
	}

	return stringData, nil
}

func idPortenClientSecretData(in *nais_io_v1.IDPortenClient, jwk jose.JSONWebKey) (map[string]string, error) {
	wellKnownURL := viper.GetString(config.DigDirIDPortenBaseURL) + "/idporten-oidc-provider/.well-known/openid-configuration"
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		secrets.IDPortenJwkKey:          string(jwkJson),
		secrets.IDPortenWellKnownURLKey: wellKnownURL,
		secrets.IDPortenClientIDKey:     in.GetStatus().GetClientID(),
		secrets.IDPortenRedirectURIKey:  string(in.Spec.RedirectURI),
	}, nil
}

func maskinportenClientSecretData(in *nais_io_v1.MaskinportenClient, jwk jose.JSONWebKey) (map[string]string, error) {
	wellKnownURL := viper.GetString(config.DigDirMaskinportenBaseURL) + "/.well-known/oauth-authorization-server"
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		secrets.MaskinportenJwkKey:          string(jwkJson),
		secrets.MaskinportenWellKnownURLKey: wellKnownURL,
		secrets.MaskinportenClientIDKey:     in.GetStatus().GetClientID(),
		secrets.MaskinportenScopesKey:       strings.Join(in.GetConsumedScopes(), " "),
	}, nil
}
