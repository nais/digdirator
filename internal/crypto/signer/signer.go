package signer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-jose/go-jose/v4"
)

const (
	SigningAlg = jose.RS256
)

var _ jose.Signer = (*ConfigurableSigner)(nil)

type (
	ByteSigner interface {
		SignBytes(payload []byte) ([]byte, error)
	}

	ConfigurableSigner struct {
		SignerOptions *jose.SignerOptions
		ByteSigner    ByteSigner
	}
)

func (s ConfigurableSigner) Options() jose.SignerOptions {
	return *s.SignerOptions
}

func (s ConfigurableSigner) Sign(payload []byte) (*jose.JSONWebSignature, error) {
	header := map[jose.HeaderKey]interface{}{
		"alg": SigningAlg,
	}
	for k, v := range s.SignerOptions.ExtraHeaders {
		header[k] = v
	}

	headerJson, err := json.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("marshalling payload: %w", err)
	}

	// encode the header and the payload
	var signable bytes.Buffer
	signable.WriteString(base64.RawURLEncoding.EncodeToString(headerJson))
	signable.WriteByte('.')
	signable.WriteString(base64.RawURLEncoding.EncodeToString(payload))

	// sign the result
	signature, err := s.ByteSigner.SignBytes(signable.Bytes())
	if err != nil {
		return nil, fmt.Errorf("signing bytes: %w", err)
	}

	// now all together to form the signed JWT in compact serialization format
	var out bytes.Buffer
	out.WriteString(signable.String())
	out.WriteByte('.')
	out.WriteString(base64.RawURLEncoding.EncodeToString(signature))

	return jose.ParseSigned(out.String(), []jose.SignatureAlgorithm{SigningAlg})
}
