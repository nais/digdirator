package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"

	"github.com/nais/digdirator/pkg/crypto"
)

const (
	CertChainPath = "cert-chain-path"
	PublicKeyPath = "public-key-path"
	Output        = "output"
)

func init() {
	// Automatically read configuration options from environment variables.
	// i.e. --cert-chain-path will be configurable using PEM2JWK_CERT_CHAIN_PATH.
	viper.SetEnvPrefix("PEM2JWK")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	flag.String(CertChainPath, "chain.pem", "The certificate chain including the certificate itself, in PEM format.")
	flag.String(PublicKeyPath, "publickey.pem", "The PKIX public key associated with the certificate, in PEM format.")
	flag.String(Output, "public.jwk", "Path to output the resulting JWK to.")

	flag.Parse()

	err := viper.BindPFlags(flag.CommandLine)
	if err != nil {
		panic(err)
	}
}

func main() {
	certificates, err := parseCertificates()
	if err != nil {
		panic(fmt.Errorf("while parsing certificates: %w", err))
	}

	publicKey, err := parsePublicKey()
	if err != nil {
		panic(fmt.Errorf("while parsing public key: %w", err))
	}

	jwk := convertToPublicJwk(certificates, publicKey)

	src, err := jwk.MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("while marshalling json: %w", err))
	}

	dst := &bytes.Buffer{}
	if err := json.Indent(dst, src, "", "  "); err != nil {
		panic(err)
	}

	outputPath := viper.GetString(Output)
	file, err := os.Create(outputPath)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = file.Write(dst.Bytes())
	if err != nil {
		panic(fmt.Errorf("writing to output: %w", err))
	}
}

func parseCertificates() ([]*x509.Certificate, error) {
	path := viper.GetString(CertChainPath)
	certChainPEM, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("while reading %s: %w", path, err)
	}

	var certDERBlock *pem.Block
	certificates := make([]*x509.Certificate, 0)
	for {
		certDERBlock, certChainPEM = pem.Decode(certChainPEM)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			certificates = append(certificates, &x509.Certificate{Raw: certDERBlock.Bytes})
		}
	}

	return certificates, nil
}

func parsePublicKey() (interface{}, error) {
	path := viper.GetString(PublicKeyPath)
	pubPEM, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("while reading %s: %w", path, err)
	}

	block, _ := pem.Decode(pubPEM)
	if block == nil {
		return nil, fmt.Errorf("while parsing PEM block containing the public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("while parsing DER encoded public key: %w", err)
	}

	return publicKey, nil
}

func convertToPublicJwk(certificates []*x509.Certificate, publicKey interface{}) jose.JSONWebKey {
	// note: RFC7517 for JWK defines the `x5t` and `x5t#S256` parameters as base64url-encoded SHA-1 and SHA256 thumbprints/digests.
	// we set these digests as-is without encoding them, as the go-jose library already does the encoding when marshalling the key.
	certSha1 := sha1.Sum(certificates[0].Raw)
	certSha256 := sha256.Sum256(certificates[0].Raw)

	// we'll use the `x5t#S256` thumbprint as the `kid`, as this can reliably be calculated using the certificate when creating JWT client assertions elsewhere (e.g. during runtime).
	keyId := crypto.X5tS256(certificates[0])

	jwk := jose.JSONWebKey{
		Key:                         publicKey,
		Algorithm:                   string(jose.RS256),
		KeyID:                       keyId,
		Use:                         "sig",
		Certificates:                certificates,
		CertificateThumbprintSHA1:   certSha1[:],
		CertificateThumbprintSHA256: certSha256[:],
	}

	return jwk
}
