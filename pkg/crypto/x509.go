package crypto

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

func ConvertPEMBlockToX509Chain(certPEMBlock []byte) ([]*x509.Certificate, error) {
	var certDERBlock *pem.Block
	certs := make([]*x509.Certificate, 0)
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
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
