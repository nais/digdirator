package crypto

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

func ConvertPEMChainToX509Chain(pemChain []byte) ([]*x509.Certificate, error) {
	var certDERBlock *pem.Block
	certs := make([]*x509.Certificate, 0)
	for {
		certDERBlock, pemChain = pem.Decode(pemChain)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(certDERBlock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("while parsing certificate: %w", err)
			}
			certs = append(certs, cert)
		}
	}
	return certs, nil
}

func ConvertX509CertificatesToX5c(certs []*x509.Certificate) []string {
	x5c := make([]string, 0)
	for _, cert := range certs {
		x5c = append(x5c, base64.StdEncoding.EncodeToString(cert.Raw))
	}
	return x5c
}

// X5tS256 creates a base64url-encoded SHA-256 thumbprint of the given input certificate, as described in RFC 7517 section 4.9, i.e. the "x5t#S256" property.
func X5tS256(cert *x509.Certificate) string {
	sha256sum := sha256.Sum256(cert.Raw)
	return base64.RawURLEncoding.EncodeToString(sha256sum[:])
}
