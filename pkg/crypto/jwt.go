package crypto

import (
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func GenerateJwt(signer jose.Signer, claims jwt.Claims) (string, error) {
	return jwt.Signed(signer).Claims(claims).CompactSerialize()
}
