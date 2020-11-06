package crypto_test

import (
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/secrets"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestMergeJwks(t *testing.T) {
	jwkString := `
{
    "p": "8x4ONWS7qhO5QsW0zaepUfUkhiqHZ9itSuFtmRPFdV2XLVifS_Y3SXKeIFvahJhR5TGy60XlvPlw35WCpU8DSOeYuDN2mqEKbOKPrmJ9SdVWSA4wDC7FdPCr1bmolJ5kHxmhmernfgfNSRP_vZhMHDzqafNXFzEsxYOAoj0CvgU",
    "kty": "RSA",
    "q": "qLMtr-IOJ6pCoFCyDLMe-i0r4bE9Bry7_mvT9doANiJM5ZLaINNu3etTH4gnud20jY3IDoBP4hPBmurCrH5Dc2owv8OwwnV0NaCKB1nFAgFcTTQTXinWadhq_MM6ddS44FuVnTWQON-nbpjeVSU9aAiVoWCa9qxZTL97p0y5AqM",
    "d": "O-qrNryRxGZEwuzabzOHjeBdpsM6M9eX9pAMYTA2SPH5y56pZkyCSNQtgoee7I2CxIKPugsEoP8L-zwhDE25blLzO1JUZq462VHOIsQd45UOJllWbwZTNVTylPJA4eQ5Eud_xquiaNYlWm-lmWKNP7m4g4JoGFm4GMIMHzP83_WQrDZj2Uv7mFUfJn93fNJlZuEphR09NrwioH5huNP7LBQRj4BjXiWPNKuq-d3joLzI2xTFjI9iL4LQiwhcYHrAi4LUjm_iX_6k1myEwjahcYYU13uSw9LG3fb_X5iHYNpBjP-DlUyVhops3Ft1PDudhiyDHhm6mPu1hj7__jfQ8Q",
    "e": "AQAB",
    "use": "sig",
    "kid": "uNSPBvLLVpJszqohjY98hn81YUwy3O5bpZX7OXwgyR0",
    "qi": "5pauqZYnEZhqBbUmRWFVY9V8fvuONpRPvvpvFo92b_woXIN8_924A-DFCA6ZfpU5beSjnbtNfhyP-tYQbtN3nTcTXBZyYff6L8_2rNmcNRa7bCxI6OHCUg64BemyXI449hz187zE20YGhedVUyUPA41qAz9so97xvFrCQqsxVCE",
    "dp": "QDiQSEpzyFmtdpYDTNAdSikXnNlfK29xV3Z1HRq77mTqqm_epJJFyIEehC2_a4dRGtomCUBNj73UszsrZ7-XfoqvLPlrOy2PM3QlwEsEDZztTdtxlcZFIr7wpWSFw7yTdiOvLJmAzSoCcGt4Av1YHZ15zsMZHmc_DG3QbQrwzoE",
    "alg": "RS256",
    "dq": "X2WTfFZUstFxA78eMFhKOCa7HdFgNSMdG-5V2j0AyZvz6A53EwD9PLkKNFaGQHDC3RlD_A9LHQkW_keq9mggNG_kSUyb9Br_MCQsaaO16EBktbOxEBqQiSI8vdqYgHFeamDf5hqYB9FRmRURBQ0eAGp6UtuSRdOIXAIcJqsYJAk",
    "n": "oDXiukopqeTly9XGmK40cnknXhME-5xwXasYI-Tr-qev7TRMAXtxFqYjH13pxoROe1hYCM1e_AG5IAZPwqfitqlDLum_9qtEfeL1P3b436GZDPeMRr1Sx_8_QIlDPp3_MnQ3C6EW9mAypF5WSL0TrkUQHlypURdBLhVmZGVQhzpp46YObOjpfyfv_kKRXrVaX4FP4VpL0scSvsJIADw9BBeat7BaM6F9GX8wiDVwT0zsyFq_91nkpMKtyHQFqyz4ANrUFIdmnLhMYA_nexPa_tzuA3QBvxZvjpjqKWlkFGrrcCE5jvl3pEu_PtNE5IaW8caUmUlRDZ1I6G9uq9gHLw"
}
`
	secretsInUse := v1.SecretList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: metav1.ListMeta{},
		Items: []v1.Secret{
			{
				Data: map[string][]byte{secrets.IDPortenJwkKey: []byte(jwkString)},
			},
			{
				Data: map[string][]byte{secrets.IDPortenJwkKey: []byte(jwkString)},
			},
			{
				Data: map[string][]byte{secrets.MaskinportenJwkKey: []byte(jwkString)},
			},
		},
	}
	newJwk, err := crypto.GenerateJwk()
	assert.NoError(t, err)

	jwks, err := crypto.MergeJwks(*newJwk, secretsInUse, secrets.IDPortenJwkKey)
	assert.NoError(t, err)

	assert.Len(t, jwks.Keys, 2, "should merge new JWK with JWKs in use without duplicates")

	assert.Equal(t, newJwk.Public(), jwks.Keys[0], "new public JWK should be the first entry in the returned JWKS")

	var jwk jose.JSONWebKey
	err = jwk.UnmarshalJSON([]byte(jwkString))
	assert.Equal(t, jwk.Public(), jwks.Keys[1], "existing JWK in JWKS should be public")
}
