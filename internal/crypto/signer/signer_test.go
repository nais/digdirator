package signer_test

import (
	libcrypto "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/nais/digdirator/internal/crypto/signer"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/stretchr/testify/assert"
)

type rsaSigner struct {
	signingKey   jose.SigningKey
	signatureAlg jose.SignatureAlgorithm
	opts         *jose.SignerOptions
}

func TestSignWithRsaSigner(t *testing.T) {
	claims := jwt.Claims{
		Issuer:    "iss",
		Audience:  []string{"yolo"},
		Expiry:    jwt.NewNumericDate(time.Now().Add(2 * time.Minute)),
		NotBefore: jwt.NewNumericDate(time.Now()),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ID:        uuid.New().String(),
	}
	privateKey, err := crypto.GenerateRSAKey()

	jwk := &jose.JSONWebKey{
		Key:       privateKey,
		KeyID:     uuid.New().String(),
		Use:       crypto.KeyUseSignature,
		Algorithm: crypto.KeyAlgorithm,
	}

	signerOpts := jose.SignerOptions{}
	signerOpts.WithType("JWT")
	jwkSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jwk.Key}, &signerOpts)
	rsaSigner := NewRsaSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, &signerOpts)

	signedWithJwk, err := jwt.Signed(jwkSigner).Claims(claims).Serialize()
	signedWithRsa, err := jwt.Signed(rsaSigner).Claims(claims).Serialize()
	assert.NoError(t, err)
	fmt.Println(signedWithJwk)
	fmt.Println(signedWithRsa)
	assert.Equal(t, signedWithJwk, signedWithRsa)
}

func NewRsaSigner(key jose.SigningKey, opts *jose.SignerOptions) jose.Signer {
	sign := &signer.ConfigurableSigner{
		SignerOptions: opts,
		ByteSigner: &rsaSigner{
			signingKey:   key,
			signatureAlg: signer.SigningAlg,
			opts:         opts,
		},
	}
	return sign
}

func (ctx *rsaSigner) SignBytes(payload []byte) ([]byte, error) {
	rng := rand.Reader
	key := ctx.signingKey.Key.(*rsa.PrivateKey)
	hashed := sha256.Sum256(payload)
	signature, err := rsa.SignPKCS1v15(rng, key, libcrypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}
	return signature, nil
}
