package crypto

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
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
	// todo - add x5c with corporate cert to enable recovation?
	//  see https://difi.github.io/felleslosninger/oidc_api_admin.html#bruk-av-asymmetrisk-n%C3%B8kkel
	jwk := &jose.JSONWebKey{
		Key:       privateKey,
		KeyID:     uuid.New().String(),
		Use:       KeyUseSignature,
		Algorithm: KeyAlgorithm,
	}
	return jwk, nil
}

func LoadJwkFromPath(path string) (*jose.JSONWebKey, error) {
	creds, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading JWK from path %s: %w", path, err)
	}

	jwk := &jose.JSONWebKey{}
	err = jwk.UnmarshalJSON(creds)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling JWK: %w", err)
	}

	return jwk, nil
}
