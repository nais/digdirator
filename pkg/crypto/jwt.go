package crypto

import (
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func GenerateJwt(signer jose.Signer, claims interface{}) (string, error) {
	return jwt.Signed(signer).Claims(claims).Serialize()
}
