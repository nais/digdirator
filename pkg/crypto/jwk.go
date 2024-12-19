package crypto

import (
	"fmt"

	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
	"github.com/nais/liberator/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
)

var ErrNoPreviousJwkFound = fmt.Errorf("no previous JWK found in secrets")

const (
	KeyUseSignature string = "sig"
	KeyAlgorithm    string = "RS256"
)

func GenerateJwk() (*jose.JSONWebKey, error) {
	privateKey, err := GenerateRSAKey()
	if err != nil {
		return nil, fmt.Errorf("generating RSA key for JWK: %w", err)
	}
	jwk := &jose.JSONWebKey{
		Key:       privateKey,
		KeyID:     uuid.New().String(),
		Use:       KeyUseSignature,
		Algorithm: KeyAlgorithm,
	}
	return jwk, nil
}

func GetPreviousJwkFromSecret(managedSecrets kubernetes.SecretLists, secretKey string) (*jose.JSONWebKey, error) {
	var newestSecret corev1.Secret

	for _, secret := range append(managedSecrets.Used.Items, managedSecrets.Unused.Items...) {
		if secret.CreationTimestamp.After(newestSecret.CreationTimestamp.Time) {
			newestSecret = secret
		}
	}

	key, err := getJWKFromSecret(newestSecret, secretKey)
	if err == nil && key != nil {
		return key, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting jwk from secret: %w", err)
	}

	return nil, ErrNoPreviousJwkFound
}
