package crypto

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/nais/liberator/pkg/kubernetes"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
)

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

func GetPreviousJwkFromSecret(managedSecrets *kubernetes.SecretLists, secretKey string) (*jose.JSONWebKey, error) {
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

	return nil, fmt.Errorf("no previous jwk found from managed secret")
}

// X5tS256 creates a base64url-encoded SHA-256 thumbprint of the given input certificate, as described in RFC 7517 section 4.9, i.e. the "x5t#S256" property.
func X5tS256(cert *x509.Certificate) string {
	sha256sum := sha256.Sum256(cert.Raw)
	return base64.RawURLEncoding.EncodeToString(sha256sum[:])
}
