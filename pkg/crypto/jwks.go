package crypto

import (
	"fmt"
	"github.com/nais/digdirator/pkg/secrets"
	"gopkg.in/square/go-jose.v2"
	v1 "k8s.io/api/core/v1"
)

func MergeJwks(jwk jose.JSONWebKey, secretsInUse v1.SecretList) (*jose.JSONWebKeySet, error) {
	keys := make([]jose.JSONWebKey, 0)
	keys = append(keys, jwk.Public())

	for _, secret := range secretsInUse.Items {
		key, err := getJWKFromSecret(secret)
		if err != nil {
			return nil, fmt.Errorf("getting key IDs from secret: %w", err)
		}
		keys = append(keys, key)
	}

	keys = unique(keys)
	return &jose.JSONWebKeySet{Keys: keys}, nil
}

func KeyIDsFromJwks(jwks *jose.JSONWebKeySet) []string {
	keyIDs := make([]string, 0)
	for _, key := range jwks.Keys {
		keyIDs = append(keyIDs, key.KeyID)
	}
	return keyIDs
}

func unique(keys []jose.JSONWebKey) []jose.JSONWebKey {
	seen := map[string]jose.JSONWebKey{}
	filteredKeys := make([]jose.JSONWebKey, 0)

	for _, key := range keys {
		if _, found := seen[key.KeyID]; !found {
			seen[key.KeyID] = key
			filteredKeys = append(filteredKeys, key)
		}
	}
	return filteredKeys
}

func getJWKFromSecret(secret v1.Secret) (jose.JSONWebKey, error) {
	jwkBytes := secret.Data[secrets.JwkKey]

	var jwk jose.JSONWebKey
	if err := jwk.UnmarshalJSON(jwkBytes); err != nil {
		return jwk, fmt.Errorf("unmarshalling jwk from secret")
	}

	return jwk, nil
}
