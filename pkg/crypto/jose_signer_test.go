package crypto

import (
    "crypto"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "fmt"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "gopkg.in/square/go-jose.v2"
    "gopkg.in/square/go-jose.v2/jwt"
    "testing"
    "time"
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
    privateKey, err := GenerateRSAKey()

    jwk := &jose.JSONWebKey{
        Key:       privateKey,
        KeyID:     uuid.New().String(),
        Use:       KeyUseSignature,
        Algorithm: KeyAlgorithm,
    }

    signerOpts := jose.SignerOptions{}
    signerOpts.WithType("JWT")
    jwkSigner,err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jwk.Key}, &signerOpts)
    rsaSigner := NewRsaSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, &signerOpts)

    signedWithJwk, err := jwt.Signed(jwkSigner).Claims(claims).CompactSerialize()
    signedWithRsa, err := jwt.Signed(rsaSigner).Claims(claims).CompactSerialize()
    assert.NoError(t, err)
    fmt.Println(signedWithJwk)
    fmt.Println(signedWithRsa)
    assert.Equal(t, signedWithJwk, signedWithRsa)
}

func NewRsaSigner(key jose.SigningKey, opts *jose.SignerOptions) jose.Signer {
    signer := &ConfigurableSigner{
        SignerOptions: opts,
        ByteSigner: &rsaSigner{
            signingKey:   key,
            signatureAlg: SigningAlg,
            opts:         opts,
        },
    }
    return signer
}

func (ctx *rsaSigner) SignBytes(payload []byte) ([]byte, error) {
    rng := rand.Reader
    key := ctx.signingKey.Key.(*rsa.PrivateKey)
    hashed := sha256.Sum256(payload)
    signature, err := rsa.SignPKCS1v15(rng, key, crypto.SHA256, hashed[:])
    if err != nil {
        return nil, err
    }
    return signature, nil
}
