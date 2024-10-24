package signer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/go-jose/go-jose/v4"
	"github.com/nais/digdirator/internal/crypto"
)

var _ ByteSigner = (*KmsByteSigner)(nil)

type KmsByteSigner struct {
	Client     *kms.KeyManagementClient
	KmsKeyPath string
}

func NewKmsSigner(ctx context.Context, kmsKeyPath string, pemChain []byte) (jose.Signer, error) {
	certs, err := crypto.ConvertPEMChainToX509Chain(pemChain)
	if err != nil {
		return nil, fmt.Errorf("converting PEM cert chain to X509 cert chain: %w", err)
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in PEM chain")
	}

	// Calculate and use the `x5t#S256` value as the `kid`
	// This must match the `kid` for the pre-registered public key at the Authorization Server, exchanged out-of-band.
	kid := crypto.X5tS256(certs[0])

	opts := &jose.SignerOptions{}
	opts.WithType("JWT")
	opts.WithHeader("kid", kid) // TODO: remove this after jwk-updater has been used to remove existing keys
	//opts.WithHeader("x5c", crypto.ConvertX509CertificatesToX5c(certs)) // TODO: uncomment after above TODO has been removed

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating key management client: %v", err)
	}

	return ConfigurableSigner{
		SignerOptions: opts,
		ByteSigner: KmsByteSigner{
			Client:     kmsClient,
			KmsKeyPath: kmsKeyPath,
		},
	}, nil
}

func (k KmsByteSigner) SignBytes(payload []byte) ([]byte, error) {
	digest := sha256.New()
	if _, err := digest.Write(payload); err != nil {
		return nil, fmt.Errorf("failed to create digest of content: %v", err)
	}

	req := &kmspb.AsymmetricSignRequest{
		Name: k.KmsKeyPath,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest.Sum(nil),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	response, err := k.Client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to sign digest: %w", err)
	}

	return response.Signature, nil
}
