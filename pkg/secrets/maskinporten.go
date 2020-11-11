package secrets

import (
	"fmt"
	v1 "github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/config"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"
	"strings"
)

const (
	MaskinportenJwkKey       = "MASKINPORTEN_CLIENT_JWK"
	MaskinportenClientID     = "MASKINPORTEN_CLIENT_ID"
	MaskinportenWellKnownURL = "MASKINPORTEN_WELL_KNOWN_URL"
	MaskinportenScopes       = "MASKINPORTEN_SCOPES"
)

func MaskinportenStringData(jwk jose.JSONWebKey, instance *v1.MaskinportenClient) (map[string]string, error) {
	wellKnownURL := viper.GetString(config.DigDirMaskinportenBaseURL) + "/.well-known/oauth-authorization-server"
	jwkJson, err := jwk.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling JWK: %w", err)
	}
	return map[string]string{
		MaskinportenJwkKey:       string(jwkJson),
		MaskinportenWellKnownURL: wellKnownURL,
		MaskinportenClientID:     instance.GetStatus().GetClientID(),
		MaskinportenScopes:       JoinToString(instance.Spec.Scopes),
	}, nil
}

func JoinToString(scopes []string) string {
	return strings.Join(scopes[:], " ")
}
