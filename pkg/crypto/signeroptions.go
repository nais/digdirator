package crypto

import (
	"fmt"

	"gopkg.in/square/go-jose.v2"
)

func SetupSignerOptions(pemChain []byte) (*jose.SignerOptions, error) {
	certs, err := ConvertPEMChainToX509Chain(pemChain)
	if err != nil {
		return nil, fmt.Errorf("converting PEM cert chain to X509 cert chain: %w", err)
	}

	// Calculate and use the `x5t#S256` value as the `kid`
	// This must match the `kid` for the pre-registered public key at the Authorization Server, exchanged out-of-band.
	kid := X5tS256(certs[0])

	signerOpts := &jose.SignerOptions{}
	signerOpts.WithType("JWT")
	signerOpts.WithHeader("kid", kid)

	return signerOpts, nil
}
