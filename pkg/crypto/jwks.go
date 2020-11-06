package crypto

import (
	"fmt"
	"gopkg.in/square/go-jose.v2"
	v1 "k8s.io/api/core/v1"
)

func MergeJwks(jwk jose.JSONWebKey, secretsInUse v1.SecretList, secretKey string) (*jose.JSONWebKeySet, error) {
	keys := make([]jose.JSONWebKey, 0)
	keys = append(keys, jwk.Public())

	for _, secret := range secretsInUse.Items {
		key, err := getJWKFromSecret(secret, secretKey)
		if err != nil {
			return nil, fmt.Errorf("getting key IDs from secret: %w", err)
		}
		if key != nil {
			keyValue := *key
			keys = append(keys, keyValue.Public())
		}
	}

	return &jose.JSONWebKeySet{Keys: unique(keys)}, nil
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
	filtered := make([]jose.JSONWebKey, 0)

	for _, key := range keys {
		if _, found := seen[key.KeyID]; !found {
			seen[key.KeyID] = key
			filtered = append(filtered, key)
		}
	}
	return filtered
}

func getJWKFromSecret(secret v1.Secret, key string) (*jose.JSONWebKey, error) {
	var jwk jose.JSONWebKey
	jwkBytes, found := secret.Data[key]
	if !found {
		return nil, nil
	}
	if err := jwk.UnmarshalJSON(jwkBytes); err != nil {
		return nil, fmt.Errorf("unmarshalling JWK from secret")
	}
	return &jwk, nil
}
