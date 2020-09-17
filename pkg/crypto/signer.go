package crypto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gopkg.in/square/go-jose.v2"
)

const SigningAlg = "RS256"

type ByteSigner interface {
	SignBytes(payload []byte) ([]byte, error)
}

type ConfigurableSigner struct {
	SignerOptions *jose.SignerOptions
	ByteSigner    ByteSigner
}

func (ctx *ConfigurableSigner) Options() jose.SignerOptions {
	return *ctx.SignerOptions
}

func (ctx *ConfigurableSigner) Sign(payload []byte) (*jose.JSONWebSignature, error) {
	obj := &jose.JSONWebSignature{}
	obj.Signatures = make([]jose.Signature, 1)

	headersAndPayload, err := serializeHeadersAndPayload(ctx.SignerOptions, payload)
	if err != nil {
		return nil, fmt.Errorf("serializing headers and payload: %w", err)
	}
	signatureInfo, err := ctx.ByteSigner.SignBytes(headersAndPayload.Bytes())
	if err != nil {
		return nil, fmt.Errorf("signing bytes: %w", err)
	}
	var output bytes.Buffer
	output.WriteString(headersAndPayload.String())
	output.WriteByte('.')
	output.WriteString(base64.RawURLEncoding.EncodeToString(signatureInfo))
	return jose.ParseSigned(output.String())
}

func SignerFromJwk(jwk *jose.JSONWebKey) (jose.Signer, error) {
	signerOpts := jose.SignerOptions{}
	signerOpts.WithType("JWT")
	signerOpts.WithHeader("x5c", extractX5c(jwk))

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: jwk.Key}, &signerOpts)
	if err != nil {
		return nil, fmt.Errorf("creating jwt signer: %v", err)
	}
	return signer, nil
}

func serializeHeadersAndPayload(opts *jose.SignerOptions, payload []byte) (bytes.Buffer, error) {
	protected := map[jose.HeaderKey]interface{}{
		"alg": SigningAlg,
	}
	for k, v := range opts.ExtraHeaders {
		protected[k] = v
	}
	serializedProtected, err := json.Marshal(protected)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("marshalling payload: %w", err)
	}
	var headersAndPayloadSerialized bytes.Buffer
	headersAndPayloadSerialized.WriteString(base64.RawURLEncoding.EncodeToString(serializedProtected))
	headersAndPayloadSerialized.WriteByte('.')
	headersAndPayloadSerialized.WriteString(base64.RawURLEncoding.EncodeToString(payload))
	return headersAndPayloadSerialized, err
}

func extractX5c(jwk *jose.JSONWebKey) []string {
	x5c := make([]string, 0)
	for _, cert := range jwk.Certificates {
		x5c = append(x5c, base64.StdEncoding.EncodeToString(cert.Raw))
	}
	return x5c
}
