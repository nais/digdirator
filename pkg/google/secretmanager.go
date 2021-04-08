package google

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
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

func (in *secretManagerClient) KeyChainMetadata(ctx context.Context, projectID, secretName, secretVersion string) ([]byte, error) {
	req := ToAccessSecretVersionRequest(projectID, secretName, secretVersion)
	secretData, err := in.GetSecretData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetching keychain metadata (secret '%s', version '%s' project ID '%s'): %w", secretName, secretVersion, projectID, err)
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

func ToAccessSecretVersionRequest(projectID, secretName, secretVersion string) *secretmanagerpb.AccessSecretVersionRequest {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secretName, secretVersion)
	return &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}
}
