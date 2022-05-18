package secrets

import (
	"fmt"
	"strings"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"gopkg.in/square/go-jose.v2"

	"github.com/nais/digdirator/pkg/config"
)

func IDPortenClientSecretData(in *nais_io_v1.IDPortenClient, jwk jose.JSONWebKey, config *config.Config) (map[string]string, error) {
	wellKnownURL := config.DigDir.IDPorten.WellKnownURL
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		IDPortenJwkKey:          string(jwkJson),
		IDPortenWellKnownURLKey: wellKnownURL,
		IDPortenClientIDKey:     in.GetStatus().GetClientID(),
		IDPortenRedirectURIKey:  string(in.Spec.RedirectURI),
	}, nil
}

func MaskinportenClientSecretData(in *nais_io_v1.MaskinportenClient, jwk jose.JSONWebKey, config *config.Config) (map[string]string, error) {
	wellKnownURL := config.DigDir.Maskinporten.WellKnownURL
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		MaskinportenJwkKey:          string(jwkJson),
		MaskinportenWellKnownURLKey: wellKnownURL,
		MaskinportenClientIDKey:     in.GetStatus().GetClientID(),
		MaskinportenScopesKey:       strings.Join(in.GetConsumedScopes(), " "),
	}, nil
}
