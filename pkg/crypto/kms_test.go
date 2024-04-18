package crypto_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
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
