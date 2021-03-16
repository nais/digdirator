package google_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/fake"
	"github.com/nais/digdirator/pkg/google"
)

var (
	payload    = readPemFile()
	ctx        = context.Background()
	projectID  = "some-project-id"
	secretName = "some-secret-name"
)

func TestToAccessSecretVersionRequest(t *testing.T) {
	projectID := "some-project"
	secretName := "some-secret"

	expected := "projects/some-project/secrets/some-secret/versions/latest"
	actual := google.ToAccessSecretVersionRequest(projectID, secretName)

	assert.Equal(t, expected, actual.GetName())
}

// This is to verify that changes to SetupSignerOptions works as expected
func TestGetKeyChainAndSetupOfSignerOptions(t *testing.T) {
	secretManagerClient := fake.NewSecretManagerClient(payload, nil)
	data, err := secretManagerClient.KeyChainMetadata(ctx, projectID, secretName)
	assert.NoError(t, err)
	assert.Equal(t, payload, data)

	signerOption, err := crypto.SetupSignerOptions(data)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(signerOption.ExtraHeaders))
}

func readPemFile() []byte {
	payload, err := ioutil.ReadFile("testdata/cert.pem")
	if err != nil {
		return nil
	}
	return payload
}
