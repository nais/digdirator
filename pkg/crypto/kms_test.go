package crypto

import (
	"crypto/rsa"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"testing"
	"time"
)

func TestSignWithKms(t *testing.T) {

	claims := jwt.Claims{
		Issuer:    "iss",
		Audience:  []string{"yolo"},
		Expiry:    jwt.NewNumericDate(time.Now().Add(2 * time.Minute)),
		NotBefore: jwt.NewNumericDate(time.Now()),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ID:        uuid.New().String(),
	}
	privateKey, err := GenerateRSAKey()

	jwk := &jose.JSONWebKey{
		Key:       privateKey,
		KeyID:     uuid.New().String(),
		Use:       KeyUseSignature,
		Algorithm: KeyAlgorithm,
	}
	jwkSigner, err := signerFromJwk(jwk)
	signedWithJwk, err := jwt.Signed(jwkSigner).Claims(claims).CompactSerialize()

	assert.NoError(t, err)

	kmsSigner, err := signerFromKms(privateKey)

	signedWithKms, err := jwt.Signed(kmsSigner).Claims(claims).CompactSerialize()
	assert.NoError(t, err)
	fmt.Println(signedWithJwk)
	fmt.Println(signedWithKms)
	assert.Equal(t, signedWithJwk, signedWithKms)
}

func signerFromKms(key *rsa.PrivateKey) (jose.Signer, error) {
	signerOpts := jose.SignerOptions{}
	signerOpts.WithType("JWT")
	return NewKmsSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, key, &signerOpts)
}

func signerFromJwk(jwk *jose.JSONWebKey) (jose.Signer, error) {
	signerOpts := jose.SignerOptions{}
	signerOpts.WithType("JWT")
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jwk.Key}, &signerOpts)
	return signer, err
}
