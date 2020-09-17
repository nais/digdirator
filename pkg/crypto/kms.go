package crypto

import (
    kms "cloud.google.com/go/kms/apiv1"
    "context"
    "crypto/sha256"
    "fmt"
    kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
    "gopkg.in/square/go-jose.v2"
    "gopkg.in/square/go-jose.v2/jwt"
)

type KmsKeyPath string

type KmsSigner struct {
    Client        *kms.KeyManagementClient
    Ctx           context.Context
    KmsKeyPath    KmsKeyPath
    SignerOptions *jose.SignerOptions
}

func NewKmsSigner(client *kms.KeyManagementClient, ctx context.Context, keyPath KmsKeyPath, signerOpts *jose.SignerOptions) jose.Signer {
    return &KmsSigner{
        Client:        client,
        Ctx:           ctx,
        KmsKeyPath:    keyPath,
        SignerOptions: signerOpts,
    }
}

func (signer *KmsSigner) Options() jose.SignerOptions {
    return *signer.SignerOptions
}

func (signer *KmsSigner) Sign(payload []byte) (*jose.JSONWebSignature, error) {



    return &jose.JSONWebSignature{}, nil
}

func createSignRequest(keyPath string, content []byte) (*kmspb.AsymmetricSignRequest, error) {
    digest := sha256.New()
    if _, err := digest.Write(content); err != nil {
        return nil, fmt.Errorf("failed to create digest of content: %v", err)
    }
    return &kmspb.AsymmetricSignRequest{
        Name: keyPath,
        Digest: &kmspb.Digest{
            Digest: &kmspb.Digest_Sha256{
                Sha256: digest.Sum(nil),
            },
        },
    }, nil
}

// TODO some versions conflict with grpc lib from kms dependency
func signWithKms(signer jose.Signer, claims jwt.Claims) (string, error) {
  /*  ctx := context.Background()
    client, err := kms.NewKeyManagementClient(ctx)
    if err != nil {
        return "", fmt.Errorf("could not create key management Client %w", err)
    }

    content, err := json.Marshal(claims)
    if err != nil {
        return "", fmt.Errorf("failed to marshal claims to bytes %w", err)
    }

    req, err := createSignRequest("keyname", content)
    if err != nil {
        return "", fmt.Errorf("could not create sign request %w".err)
    }

    result, err := client.AsymmetricSign(ctx, req)
    if err != nil {
        return "", fmt.Errorf("failed to sign digest: %w", err)
    }

    signedDigest := result.Signature

    jws := jose.JSONWebSignature{
        content,
        []jose.Signature{

        }
    }*/
    return "", nil
    //return jwt.Signed(signer).Claims(claims).CompactSerialize()
}
