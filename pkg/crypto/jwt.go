package crypto

import (
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func GenerateJwt(signer jose.Signer, claims interface{}) (string, error) {
	return jwt.Signed(signer).Claims(claims).CompactSerialize()
}
