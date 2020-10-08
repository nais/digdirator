package crypto

import (
	"crypto/sha256"
	"fmt"
	"github.com/nais/digdirator/pkg/config"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
)

func SetupSignerOptions(cfg *config.Config) (*jose.SignerOptions, error) {
	path := cfg.DigDir.Auth.CertChainPath
	certPEMBlock, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading PEM cert chain from path %s: %w", path, err)
	}
	certs, err := ConvertPEMBlockToX509Chain(certPEMBlock)
	x5c := ConvertX509CertificatesToX5c(certs)
	kid := sha256.Sum256(certs[0].Raw)

	signerOpts := &jose.SignerOptions{}
	signerOpts.WithType("JWT")
	signerOpts.WithHeader("x5c", x5c)
	signerOpts.WithHeader("kid", string(kid[:]))

	return signerOpts, nil
}
