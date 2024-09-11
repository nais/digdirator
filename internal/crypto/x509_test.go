package crypto_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/nais/digdirator/internal/crypto"
	"github.com/stretchr/testify/assert"
)

var (
	rootSubject = pkix.Name{
		Country:      []string{"NO"},
		Organization: []string{"ACME AS 987654321"},
		CommonName:   "ACME AS Root CA",
	}
	intermediateSubject = pkix.Name{
		Country:      []string{"NO"},
		Organization: []string{"ACME AS 987654321"},
		CommonName:   "ACME AS Intermediate CA",
	}
	certificateSubject = pkix.Name{
		Country:      []string{"NO"},
		Organization: []string{"NAIS Price AS 987654321"},
		CommonName:   "NAIS Price AS",
	}
)

func TestConvertPEMBlockToX509Chain(t *testing.T) {
	certChain, err := generateCertChain()
	assert.NoError(t, err)

	pemChainBytes, err := pemChainToBytes(certChain)
	assert.NoError(t, err)

	certs, err := crypto.ConvertPEMChainToX509Chain(pemChainBytes)
	assert.NoError(t, err)

	assert.Len(t, certs, 3)
	assert.Equal(t, "CN=NAIS Price AS,O=NAIS Price AS 987654321,C=NO", certs[0].Subject.String())
	assert.Equal(t, "CN=ACME AS Intermediate CA,O=ACME AS 987654321,C=NO", certs[1].Subject.String())
	assert.Equal(t, "CN=ACME AS Root CA,O=ACME AS 987654321,C=NO", certs[2].Subject.String())
}

func TestConvertX509CertificatesToX5c(t *testing.T) {
	certChain, err := generateCertChain()
	assert.NoError(t, err)

	x5c := crypto.ConvertX509CertificatesToX5c(certChain)
	assert.Len(t, x5c, 3)

	assertPemStringsEqualsX5cStrings(t, x5c, certChain)
}

func assertPemStringsEqualsX5cStrings(t *testing.T, x5c []string, certs []*x509.Certificate) {
	certToFormattedString := func(cert *x509.Certificate) (string, error) {
		const prefix = "-----BEGIN CERTIFICATE-----"
		const suffix = "-----END CERTIFICATE-----"

		b := bytes.Buffer{}
		if err := encodeCertToPem(cert, &b); err != nil {
			return "", err
		}

		var certString = b.String()
		certString = strings.ReplaceAll(certString, "\n", "")
		certString = strings.TrimPrefix(certString, prefix)
		certString = strings.TrimSuffix(certString, suffix)

		return certString, nil
	}

	for i, x5 := range x5c {
		certString, err := certToFormattedString(certs[i])
		assert.NoError(t, err)
		assert.Equal(t, certString, x5)
		for j := range certs {
			if i != j {
				assert.NotEqual(t, certString, x5c[j])
			}
		}
	}
}

func encodeCertToPem(cert *x509.Certificate, out io.Writer) error {
	if err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
		return err
	}
	return nil
}

func pemChainToBytes(certs []*x509.Certificate) ([]byte, error) {
	b := bytes.Buffer{}
	for _, cert := range certs {
		if err := encodeCertToPem(cert, &b); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}

func generateCertChain() ([]*x509.Certificate, error) {
	rootTemplate := certificateTemplate(rootSubject)
	intermediateTemplate := certificateTemplate(intermediateSubject)
	certificateTemplate := certificateTemplate(certificateSubject)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA keypair: %w", err)
	}

	root, err := generateCertificate(rootTemplate, rootTemplate, privateKey)
	if err != nil {
		return nil, err
	}

	intermediate, err := generateCertificate(intermediateTemplate, root, privateKey)
	if err != nil {
		return nil, err
	}

	certificate, err := generateCertificate(certificateTemplate, intermediate, privateKey)
	if err != nil {
		return nil, err
	}

	certs := []*x509.Certificate{
		certificate, intermediate, root,
	}

	return certs, nil
}

func generateCertificate(template, parent *x509.Certificate, privateKey *rsa.PrivateKey) (*x509.Certificate, error) {
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parent, privateKey.Public(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate the certificate for key: %w", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from DER data: %w", err)
	}
	return cert, nil
}

func certificateTemplate(subject pkix.Name) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
}
