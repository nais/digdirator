package google_test

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2"

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

func TestParseSecretPath(t *testing.T) {
	expected := "projects/some-project/secrets/some-secret/versions/latest"
	err := google.ParseSecretPath(expected)
	assert.NoError(t, err)

	errPath1 := "some-project/secrets/some-secret/versions/latest"
	err = google.ParseSecretPath(errPath1)
	assert.ErrorContains(t, err, "secret path must be 6 characters long")

	errPath2 := "kk/some-project/secrets/some-secret/versions/latest"
	err = google.ParseSecretPath(errPath2)
	assert.ErrorContains(t, err, "secret path must start with 'projects/'")
}

// This is to verify that changes to SetupSignerOptions works as expected
func TestGetKeyChainAndSetupOfSignerOptions(t *testing.T) {
	secretManagerClient := fake.NewSecretManagerClient(payload, nil)
	data, err := secretManagerClient.KeyChainMetadata(ctx, projectID, secretName)
	assert.NoError(t, err)
	assert.Equal(t, payload, data)

	signerOption, err := crypto.SetupSignerOptions(data)
	assert.NoError(t, err)
	assert.Equal(t, jose.ContentType("JWT"), signerOption.ExtraHeaders["typ"])
	assert.Equal(t, "PqkzrnxnIaW1x5siEQZzNp3efeif3rO4ndqW6B4B-tY", signerOption.ExtraHeaders["kid"])
	assert.Equal(t, 2, len(signerOption.ExtraHeaders))
}

func readPemFile() []byte {
	payload, err := ioutil.ReadFile("testdata/cert.pem")
	if err != nil {
		return nil
	}
	return payload
}
