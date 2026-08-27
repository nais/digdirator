package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProviders(t *testing.T) {
	tests := []struct {
		name     string
		features Features
		wantErr  string
	}{
		{
			name:    "all providers disabled",
			wantErr: "at least one provider must be enabled",
		},
		{
			name:     "ID-porten URL missing",
			features: Features{IDPorten: true},
			wantErr:  DigDirIDPortenWellKnownURL,
		},
		{
			name:     "Ansattporten URL missing",
			features: Features{Ansattporten: true},
			wantErr:  DigDirAnsattportenWellKnownURL,
		},
		{
			name:     "Maskinporten URL missing",
			features: Features{Maskinporten: true},
			wantErr:  DigDirMaskinportenWellKnownURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Features: tt.features}
			err := cfg.Validate(nil)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateAcceptsEnabledProviderWithURL(t *testing.T) {
	cfg := Config{
		Features: Features{Ansattporten: true},
		DigDir: DigDir{
			Admin:        Admin{BaseURL: "https://example.com"},
			Ansattporten: Ansattporten{WellKnownURL: "https://example.com/.well-known/openid-configuration"},
		},
	}

	require.NoError(t, cfg.Validate(nil))
}
