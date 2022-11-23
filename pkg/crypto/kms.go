package crypto

import (
	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/nais/digdirator/pkg/config"
	"gopkg.in/square/go-jose.v2"
	"time"
)

type KmsKeyPath string

type KmsOptions struct {
	Client    *kms.KeyManagementClient
	Ctx       context.Context
	KmsConfig config.KMS
}

func (k KmsOptions) keyPath() KmsKeyPath {
	return KmsKeyPath(k.KmsConfig.KeyPath)
}

type KmsByteSigner struct {
	Client        *kms.KeyManagementClient
	Ctx           context.Context
	SignerOptions *jose.SignerOptions
	KmsKeyPath    KmsKeyPath
}

func NewKmsSigner(kms *KmsOptions, opts *jose.SignerOptions) (jose.Signer, error) {
	return ConfigurableSigner{
		SignerOptions: opts,
		ByteSigner: KmsByteSigner{
			Client:        kms.Client,
			Ctx:           kms.Ctx,
			KmsKeyPath:    kms.keyPath(),
			SignerOptions: opts,
		},
	}, nil
}

func (k KmsByteSigner) SignBytes(payload []byte) ([]byte, error) {
	req, err := createSignRequest(k.KmsKeyPath, payload)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(k.Ctx, 1*time.Minute)
	defer cancel()

	response, err := k.Client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to sign digest: %w", err)
	}
	return response.Signature, nil
}

func createSignRequest(keyPath KmsKeyPath, content []byte) (*kmspb.AsymmetricSignRequest, error) {
	digest := sha256.New()
	if _, err := digest.Write(content); err != nil {
		return nil, fmt.Errorf("failed to create digest of content: %v", err)
	}
	return &kmspb.AsymmetricSignRequest{
		Name: string(keyPath),
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest.Sum(nil),
			},
		},
	}, nil
}
