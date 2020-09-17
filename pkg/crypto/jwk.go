package crypto

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/square/go-jose.v2"
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
