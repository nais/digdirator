package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

func NewRSAKeyPair() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating RSA keypair: %w", err)
	}
	return privateKey, nil
}
