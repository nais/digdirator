package crypto

import (
    "fmt"
    "github.com/google/uuid"
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

    signerOpts := jose.SignerOptions{}
    signerOpts.WithType("JWT")
    fmt.Println(claims)
}
