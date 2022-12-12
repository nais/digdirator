package crypto

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"gopkg.in/square/go-jose.v2"

	"github.com/nais/digdirator/pkg/config"
)

type KmsKeyPath string

type KmsOptions struct {
	Client    *kms.KeyManagementClient
	Ctx       context.Context
	KmsConfig config.KMS
}

func (k KmsOptions) parseKeyPath() (KmsKeyPath, error) {
	splitKmsPath := strings.Split(k.KmsConfig.KeyPath, "/")

	if len(splitKmsPath) == 0 {
		return "", fmt.Errorf("kms key path is empty")
	}

	if len(splitKmsPath) != 10 {
		return "", fmt.Errorf("kms key path must be 10 characters long")
	}

	if !strings.HasPrefix(k.KmsConfig.KeyPath, "projects/") {
		return "", fmt.Errorf("kms key path must start with 'projects/'")
	}

	if !strings.Contains(k.KmsConfig.KeyPath, "/locations/") {
		return "", fmt.Errorf("kms key path must contain '/locations/'")
	}

	if !strings.Contains(k.KmsConfig.KeyPath, "/keyRings/") {
		return "", fmt.Errorf("kms key path must contain '/keyRings/'")
	}

	if !strings.Contains(k.KmsConfig.KeyPath, "/cryptoKeys/") {
		return "", fmt.Errorf("kms key path must contain '/cryptoKeys/'")
	}

	if !strings.Contains(k.KmsConfig.KeyPath, "/cryptoKeyVersions/") {
		return "", fmt.Errorf("kms key path must contain '/cryptoKeyVersions/'")
	}

	return KmsKeyPath(k.KmsConfig.KeyPath), nil
}

type KmsByteSigner struct {
	Client        *kms.KeyManagementClient
	Ctx           context.Context
	SignerOptions *jose.SignerOptions
	KmsKeyPath    KmsKeyPath
}

func NewKmsSigner(certChain []byte, kmsConfig config.KMS, ctx context.Context) (jose.Signer, error) {
	signerOpts, err := SetupSignerOptions(certChain)
	if err != nil {
		return nil, fmt.Errorf("setting up signer options: %v", err)
	}

	kmsCtx := ctx
	kmsClient, err := kms.NewKeyManagementClient(kmsCtx)
	if err != nil {
		return nil, fmt.Errorf("error creating key management client: %v", err)
	}

	return newConfigurableSigner(&KmsOptions{
		Client:    kmsClient,
		Ctx:       kmsCtx,
		KmsConfig: kmsConfig,
	}, signerOpts)
}

func newConfigurableSigner(kms *KmsOptions, opts *jose.SignerOptions) (jose.Signer, error) {
	kmsPath, err := kms.parseKeyPath()
	if err != nil {
		return nil, err
	}

	return ConfigurableSigner{
		SignerOptions: opts,
		ByteSigner: KmsByteSigner{
			Client:        kms.Client,
			Ctx:           kms.Ctx,
			KmsKeyPath:    kmsPath,
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
