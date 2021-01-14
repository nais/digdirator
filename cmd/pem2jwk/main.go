package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"strings"
)

const (
	CertChainPath = "cert-chain-path"
	PublicKeyPath = "public-key-path"
)

func init() {
	// Automatically read configuration options from environment variables.
	// i.e. --cert-chain-path will be configurable using PEM2JWK_CERT_CHAIN_PATH.
	viper.SetEnvPrefix("PEM2JWK")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	flag.String(CertChainPath, "chain.pem", "The certificate chain including the certificate itself, in PEM format.")
	flag.String(PublicKeyPath, "publickey.pem", "The public key associated with the certificate, in PEM format.")

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

	json, err := jwk.MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("while marshalling json: %w", err))
	}

	fmt.Println(string(json))
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
	x5tSHA1 := sha1.Sum(certificates[0].Raw)
	x5tSHA256 := sha256.Sum256(certificates[0].Raw)
	keyId := base64.RawURLEncoding.EncodeToString(x5tSHA256[:])

	jwk := jose.JSONWebKey{
		Key:                         publicKey,
		KeyID:                       keyId,
		Use:                         "sig",
		Certificates:                certificates,
		CertificateThumbprintSHA1:   x5tSHA1[:],
		CertificateThumbprintSHA256: x5tSHA256[:],
	}

	return jwk
}
