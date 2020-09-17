package crypto

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
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
    signatureInfo, err := ctx.ByteSigner.SignBytes(headersAndPayload.Bytes())
    if err != nil {
        return nil, err
    }
    var output bytes.Buffer
    output.WriteString(headersAndPayload.String())
    output.WriteByte('.')
    output.WriteString(base64.RawURLEncoding.EncodeToString(signatureInfo))
    return jose.ParseSigned(output.String())
}


func serializeHeadersAndPayload(opts *jose.SignerOptions, payload []byte) (bytes.Buffer, error) {
    protected := map[jose.HeaderKey]interface{}{
        "alg": SigningAlg,
    }
    for k, v := range opts.ExtraHeaders {
        protected[k] = v
    }
    serializedProtected, err := json.Marshal(protected)
    var headersAndPayloadSerialized bytes.Buffer
    headersAndPayloadSerialized.WriteString(base64.RawURLEncoding.EncodeToString(serializedProtected))
    headersAndPayloadSerialized.WriteByte('.')
    headersAndPayloadSerialized.WriteString(base64.RawURLEncoding.EncodeToString(payload))
    return headersAndPayloadSerialized, err
}
