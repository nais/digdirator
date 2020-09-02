package crypto

import (
	"encoding/base64"
	"fmt"
	"gopkg.in/square/go-jose.v2"
)

func SignerFromJwk(jwk *jose.JSONWebKey) (jose.Signer, error) {
	signerOpts := jose.SignerOptions{}
	signerOpts.WithType("JWT")
	signerOpts.WithHeader("x5c", extractX5c(jwk))

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jwk.Key}, &signerOpts)
	if err != nil {
		return nil, fmt.Errorf("creating jwt signer: %v", err)
	}
	return signer, nil
}

func extractX5c(jwk *jose.JSONWebKey) []string {
	x5c := make([]string, 0)
	for _, cert := range jwk.Certificates {
		x5c = append(x5c, base64.StdEncoding.EncodeToString(cert.Raw))
	}
	return x5c
}
