package fake

import (
	"context"
)

type secretManagerClientImpl struct {
	data []byte
	err  error
}

func (s *secretManagerClientImpl) GetSecretData(context.Context, string, string) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s *secretManagerClientImpl) KeyChainMetadata(ctx context.Context, projectID string, secretName string) ([]byte, error) {
	byteData, err := s.GetSecretData(ctx, projectID, secretName)
	if err != nil {
		return nil, s.err
	}
	return byteData, nil
}

func NewSecretManagerClient(data []byte, err error) *secretManagerClientImpl {
	return &secretManagerClientImpl{data: data, err: err}
}
