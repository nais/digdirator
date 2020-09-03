package crypto

import (
	"gopkg.in/square/go-jose.v2"
)

// todo
func MergeJwks(jwk jose.JSONWebKey, jwksInUse jose.JSONWebKeySet) jose.JSONWebKeySet {
	// fetch existing JWKS from ID-porten
	// filter for JWKs in use
	// merge new public JWK with result
	return jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			jwk.Public(),
		}}
}
