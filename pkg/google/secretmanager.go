package google

import (
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"context"
	"fmt"
)

type SecretManagerClient interface {
	GetSecretData(ctx context.Context, projectID, secretName string) ([]byte, error)
	KeyChainMetadata(ctx context.Context, projectID string, secretName string) (string, error)
}

type secretManagerClient struct {
	*secretmanager.Client
}

func NewSecretManagerClient(ctx context.Context) (*secretManagerClient, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating secret manager client: %w", err)
	}
	return &secretManagerClient{client}, nil
}

func (in *secretManagerClient) KeyChainMetadata(ctx context.Context, certChainPath string) ([]byte, error) {
	if err := ParseSecretPath(certChainPath); err != nil {
		return nil, err
	}

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: certChainPath,
	}

	secretData, err := in.GetSecretData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetching keychain metadata (secret path '%s'): %w", certChainPath, err)
	}
	return secretData, nil
}

func (in *secretManagerClient) ClientIdMetadata(ctx context.Context, secretPath string) ([]byte, error) {
	if err := ParseSecretPath(secretPath); err != nil {
		return nil, err
	}

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretPath,
	}

	secretData, err := in.GetSecretData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetching client-id metadata (secret path '%s'): %w", secretPath, err)
	}
	return secretData, nil
}

func (in *secretManagerClient) GetSecretData(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest) ([]byte, error) {
	result, err := in.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("while accessing secretmanager: %w", err)
	}
	return result.Payload.Data, nil
}
