package crypto

import (
	"github.com/google/uuid"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
)

const (
	KeyUseSignature string = "sig"
)

func GenerateJwk() (*jose.JSONWebKey, error) {
	privateKey, err := NewRSAKeyPair()
	if err != nil {
		return nil, err
	}
	jwk := &jose.JSONWebKey{
		Key:       privateKey,
		KeyID:     uuid.New().String(),
		Use:       KeyUseSignature,
		Algorithm: "RS256",
	}
	return jwk, nil
}

func LoadJwkFromPath(path string) (*jose.JSONWebKey, error) {
	creds, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	jwk := &jose.JSONWebKey{}
	err = jwk.UnmarshalJSON(creds)
	if err != nil {
		return nil, err
	}

	return jwk, nil
}
