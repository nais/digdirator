package crypto

import (
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
)

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
