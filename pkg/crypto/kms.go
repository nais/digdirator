package crypto

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type kmsSigner struct {
	keyName       string
	kid           string
	signingKey    jose.SigningKey
	rsaPrivateKey *rsa.PrivateKey
	signatureAlg  jose.SignatureAlgorithm
	opts          *jose.SignerOptions
}

func NewKmsSigner(signingKey jose.SigningKey, rsaPrivateKey *rsa.PrivateKey, opts *jose.SignerOptions) (jose.Signer, error) {
	signer := &kmsSigner{
		keyName:       "yolo",
		kid:           "somekid",
		signingKey:    signingKey,
		rsaPrivateKey: rsaPrivateKey,
		signatureAlg:  "RS256",
		opts:          opts,
	}
	return signer, nil
}

func (ctx *kmsSigner) Options() jose.SignerOptions {
	return *ctx.opts
}

func (ctx *kmsSigner) Sign(payload []byte) (*jose.JSONWebSignature, error) {
	obj := &jose.JSONWebSignature{}
	obj.Signatures = make([]jose.Signature, 1)

	protected := map[jose.HeaderKey]interface{}{
		"alg": string(ctx.signatureAlg),
		//"kid": ctx.kid,
	}

	for k, v := range ctx.opts.ExtraHeaders {
		protected[k] = v
	}

	serializedProtected, err := json.Marshal(protected)

	var input bytes.Buffer
	input.WriteString(base64.RawURLEncoding.EncodeToString(serializedProtected))
	input.WriteByte('.')
	input.WriteString(base64.RawURLEncoding.EncodeToString(payload))

	signatureInfo, err := ctx.signBytes(input.Bytes())
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	output.WriteString(input.String())
	output.WriteByte('.')
	output.WriteString(base64.RawURLEncoding.EncodeToString(signatureInfo))
	return jose.ParseSigned(output.String())

	/*signatureInfo.Protected = jose.Header{
	      KeyID:        ctx.kid,
	      Algorithm:    string(ctx.signatureAlg),
	      ExtraHeaders: ctx.opts.ExtraHeaders,
	  }
	  obj.Signatures[0] = signatureInfo
	  return &jose.JSONWebSignature{}, nil*/
}

// TODO switch out rsa.SignPKCS1v15 with Google KMS provider
func (ctx *kmsSigner) signBytes(bytes []byte) ([]byte, error) {
	rng := rand.Reader
	hashed := sha256.Sum256(bytes)
	signature, err := rsa.SignPKCS1v15(rng, ctx.rsaPrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func createSignRequest(keyName string, content []byte) (*kmspb.AsymmetricSignRequest, error) {
	digest := sha256.New()
	if _, err := digest.Write(content); err != nil {
		return nil, fmt.Errorf("failed to create digest of content: %v", err)
	}
	return &kmspb.AsymmetricSignRequest{
		Name: keyName,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest.Sum(nil),
			},
		},
	}, nil
}

// TODO some versions conflict with grpc lib from kms dependency
func signWithKms(signer jose.Signer, claims jwt.Claims) (string, error) {
	/*ctx := context.Background()
	  client, err := kms.NewKeyManagementClient(ctx)
	  if err != nil {
	      return "", fmt.Errorf("could not create key management client %w", err)
	  }

	  content, err := json.Marshal(claims)
	  if err != nil {
	      return "", fmt.Errorf("failed to marshal claims to bytes %w", err)
	  }

	  req, err := createSignRequest("keyname", content)
	  if err != nil {
	      return "", fmt.Errorf("could not create sign request %w". err)
	  }*/

	/*result, err := client.AsymmetricSign(ctx, req)
	  if err != nil {
	      return "", fmt.Errorf("failed to sign digest: %w", err)
	  }*/

	//signedDigest := result.Signature

	/*jws := jose.JSONWebSignature{
	    content,
	    []jose.Signature {

	    }
	}*/
	return "", nil
	//return jwt.Signed(signer).Claims(claims).CompactSerialize()
}
