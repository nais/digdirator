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

func (in *secretManagerClient) KeyChainMetadata(ctx context.Context, projectID string, secretName string) ([]byte, error) {
	secretData, err := in.GetSecretData(ctx, projectID, secretName)
	if err != nil {
		return nil, fmt.Errorf("while accessing secretmanager: %w", err)
	}
	return secretData, nil
}

func (in *secretManagerClient) GetSecretData(ctx context.Context, projectID, secretName string) ([]byte, error) {
	req := ToAccessSecretVersionRequest(projectID, secretName)
	result, err := in.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("projectID %s secret version: %v", projectID, err)
	}
	return result.Payload.Data, nil
}

func ToAccessSecretVersionRequest(projectID, secretName string) *secretmanagerpb.AccessSecretVersionRequest {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName)
	return &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}
}
