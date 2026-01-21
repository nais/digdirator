package secrets

import (
	"fmt"
	"strings"

	"github.com/go-jose/go-jose/v4"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/digdirator/pkg/config"
)

func IDPortenClientSecretData(in *nais_io_v1.IDPortenClient, jwk jose.JSONWebKey, config *config.Config) (map[string]string, error) {
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}

	redirectURI := func() string {
		if in.Spec.RedirectURI != "" {
			return string(in.Spec.RedirectURI)
		}

		if len(in.Spec.RedirectURIs) > 0 {
			return string(in.Spec.RedirectURIs[0])
		}

		return ""
	}

	if err := config.DigDir.IDPorten.Metadata.Validate(config.DigDir.IDPorten.WellKnownURL); err != nil {
		return nil, fmt.Errorf("validating ID-porten metadata: %w", err)
	}

	return map[string]string{
		IDPortenJwkKey:           string(jwkJson),
		IDPortenWellKnownURLKey:  config.DigDir.IDPorten.WellKnownURL,
		IDPortenClientIDKey:      in.GetStatus().ClientID,
		IDPortenRedirectURIKey:   redirectURI(),
		IDPortenIssuerKey:        config.DigDir.IDPorten.Metadata.Issuer,
		IDPortenJwksUriKey:       config.DigDir.IDPorten.Metadata.JwksURI,
		IDPortenTokenEndpointKey: config.DigDir.IDPorten.Metadata.TokenEndpoint,
	}, nil
}

func MaskinportenClientSecretData(in *nais_io_v1.MaskinportenClient, jwk jose.JSONWebKey, config *config.Config) (map[string]string, error) {
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}

	scopes := make([]string, len(in.Spec.Scopes.ConsumedScopes))
	for i, scope := range in.Spec.Scopes.ConsumedScopes {
		scopes[i] = scope.Name
	}

	if err := config.DigDir.Maskinporten.Metadata.Validate(config.DigDir.Maskinporten.WellKnownURL); err != nil {
		return nil, fmt.Errorf("validating Maskinporten metadata: %w", err)
	}

	return map[string]string{
		MaskinportenJwkKey:           string(jwkJson),
		MaskinportenWellKnownURLKey:  config.DigDir.Maskinporten.WellKnownURL,
		MaskinportenClientIDKey:      in.GetStatus().ClientID,
		MaskinportenScopesKey:        strings.Join(scopes, " "),
		MaskinportenIssuerKey:        config.DigDir.Maskinporten.Metadata.Issuer,
		MaskinportenJwksUriKey:       config.DigDir.Maskinporten.Metadata.JwksURI,
		MaskinportenTokenEndpointKey: config.DigDir.Maskinporten.Metadata.TokenEndpoint,
	}, nil
}
