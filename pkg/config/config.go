package config

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/nais/liberator/pkg/oauth"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	MetricsAddr     string   `json:"metrics-address"`
	ClusterName     string   `json:"cluster-name"`
	DevelopmentMode bool     `json:"development-mode"`
	DigDir          DigDir   `json:"digdir"`
	Features        Features `json:"features"`
}

type DigDir struct {
	Admin        Admin        `json:"admin"`
	IDPorten     IDPorten     `json:"idporten"`
	Maskinporten Maskinporten `json:"maskinporten"`
}

type Admin struct {
	BaseURL string `json:"base-url"`
}

type IDPorten struct {
	CertificateChain CertificateChain `json:"certificate-chain"`
	ClientID         string           `json:"client-id"`
	KMS              KMS              `json:"kms"`
	Scopes           string           `json:"scopes"`
	WellKnownURL     string           `json:"well-known-url"`
	Metadata         oauth.MetadataOpenID
}

type Maskinporten struct {
	CertificateChain CertificateChain `json:"certificate-chain"`
	ClientID         string           `json:"client-id"`
	KMS              KMS              `json:"kms"`
	Scopes           string           `json:"scopes"`
	WellKnownURL     string           `json:"well-known-url"`
	Metadata         oauth.MetadataOAuth
}

type KMS struct {
	Key       string `json:"key"`
	KeyRing   string `json:"key-ring"`
	Location  string `json:"location"`
	ProjectID string `json:"project-id"`
	Version   string `json:"version"`
}

type CertificateChain struct {
	ProjectID string `json:"project-id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

type Features struct {
	Maskinporten bool `json:"maskinporten"`
}

const (
	MetricsAddress                     = "metrics-address"
	ClusterName                        = "cluster-name"
	DevelopmentMode                    = "development-mode"
	DigDirAdminBaseURL                 = "digdir.admin.base-url"
	DigDirIDportenClientID             = "digdir.idporten.client-id"
	DigDirIDportenScopes               = "digdir.idporten.scopes"
	DigDirIDportenCertChainName        = "digdir.idporten.certificate-chain.name"
	DigDirIDportenCertChainVersion     = "digdir.idporten.certificate-chain.version"
	DigDirMaskinportenClientID         = "digdir.maskinporten.client-id"
	DigDirMaskinportenScopes           = "digdir.maskinporten.scopes"
	DigDirMaskinportenCertChainName    = "digdir.maskinporten.certificate-chain.name"
	DigDirMaskinportenCertChainVersion = "digdir.maskinporten.certificate-chain.version"
	DigDirIDportenKmsKey               = "digdir.idporten.kms.key"
	DigdirIDportenKmsKeyRing           = "digdir.idporten.kms.key-ring"
	DigdirIDportenKmsLocation          = "digdir.idporten.kms.location"
	DigdirIDportenKmsProjectID         = "digdir.idporten.kms.project-id"
	DigdirIDportenKmsVersion           = "digdir.idporten.kms.version"
	DigDirMaskinportenKmsKey           = "digdir.maskinporten.kms.key"
	DigdirMaskinportenKmsKeyRing       = "digdir.maskinporten.kms.key-ring"
	DigdirMaskinportenKmsLocation      = "digdir.maskinporten.kms.location"
	DigdirMaskinportenKmsProjectID     = "digdir.maskinporten.kms.project-id"
	DigdirMaskinportenKmsVersion       = "digdir.maskinporten.kms.version"
	DigDirIDPortenWellKnownURL         = "digdir.idporten.well-known-url"
	DigDirMaskinportenWellKnownURL     = "digdir.maskinporten.well-known-url"
	FeaturesMaskinporten               = "features.maskinporten"
)

func init() {
	// Automatically read configuration options from environment variables.
	// e.g. --digdir.auth.jwk will be configurable using DIGDIRATOR_DIGDIR_AUTH_JWK.
	viper.SetEnvPrefix("DIGDIRATOR")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	// Read configuration file from working directory and/or /etc.
	// File formats supported include JSON, TOML, YAML, HCL, envfile and Java properties config files
	viper.SetConfigName("digdirator")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc")

	flag.String(MetricsAddress, ":8080", "The address the metric endpoint binds to.")
	flag.String(ClusterName, "", "The cluster in which this application should run.")
	flag.String(DevelopmentMode, "false", "Toggle for development mode.")
	flag.String(DigDirAdminBaseURL, "", "Base URL endpoint for interacting with Digdir Client Registration API")
	flag.String(DigDirIDportenClientID, "", "Client ID / issuer for JWT assertion when authenticating to DigDir.")
	flag.String(DigDirIDportenKmsKey, "", "IDporten Google KmsConfig key name used to sign JWT assertion when authenticating to DigDir.")
	flag.String(DigdirIDportenKmsKeyRing, "", "IDporten Google KmsConfig key ring name.")
	flag.String(DigdirIDportenKmsLocation, "", "IDporten Google KmsConfig key location.")
	flag.String(DigdirIDportenKmsProjectID, "", "IDporten Google KmsConfig project ID.")
	flag.String(DigdirIDportenKmsVersion, "", "IDporten Google KmsConfig key version.")
	flag.String(DigDirIDportenCertChainName, "", "Secret name in Google Secret Manager to PEM file containing certificate chain for authenticating to DigDir.")
	flag.String(DigDirIDportenCertChainVersion, "", "Secret version for the secret in Google Secret Manager.")
	flag.String(DigDirMaskinportenClientID, "", "Client ID / issuer for JWT assertion when authenticating to DigDir.")
	flag.String(DigDirMaskinportenKmsKey, "", "Maskinporten Google KmsConfig key name used to sign JWT assertion when authenticating to DigDir.")
	flag.String(DigdirMaskinportenKmsKeyRing, "", "Maskinporten Google KmsConfig key ring name.")
	flag.String(DigdirMaskinportenKmsLocation, "", "Maskinporten Google KmsConfig key location.")
	flag.String(DigdirMaskinportenKmsProjectID, "", "Maskinporten Google KmsConfig project ID.")
	flag.String(DigdirMaskinportenKmsVersion, "", "Maskinporten Google KmsConfig key version.")
	flag.String(DigDirMaskinportenCertChainName, "", "Secret name in Google Secret Manager to PEM file containing certificate chain for authenticating to DigDir.")
	flag.String(DigDirMaskinportenCertChainVersion, "", "Secret version for the secret in Google Secret Manager.")
	flag.String(DigDirMaskinportenScopes, "", "List of scopes for JWT assertion when authenticating to DigDir with Maskinporten.")
	flag.String(DigDirIDportenScopes, "", "List of scopes for JWT assertion when authenticating to DigDir with IDporten.")
	flag.String(DigDirMaskinportenWellKnownURL, "", "URL to Maskinporten well-known discovery metadata document.")
	flag.String(DigDirIDPortenWellKnownURL, "", "URL to ID-porten well-known discovery metadata document.")
	flag.Bool(FeaturesMaskinporten, false, "Feature toggle for maskinporten")
}

// Print out all configuration options except secret stuff.
func (c Config) Print(redacted []string) {
	ok := func(key string) bool {
		for _, forbiddenKey := range redacted {
			if forbiddenKey == key {
				return false
			}
		}
		return true
	}

	var keys sort.StringSlice = viper.AllKeys()

	keys.Sort()
	for _, key := range keys {
		if ok(key) {
			log.Printf("%s: %s", key, viper.GetString(key))
		} else {
			log.Printf("%s: ***REDACTED***", key)
		}
	}
}

func (c Config) Validate(required []string) error {
	present := func(key string) bool {
		for _, requiredKey := range required {
			if requiredKey == key {
				return len(viper.GetString(requiredKey)) > 0
			}
		}
		return true
	}
	var keys sort.StringSlice = viper.AllKeys()
	errs := make([]string, 0)

	keys.Sort()
	for _, key := range keys {
		if !present(key) {
			errs = append(errs, key)
		}
	}

	for _, key := range errs {
		log.Printf("required key '%s' not configured", key)
	}
	if len(errs) > 0 {
		return errors.New("missing configuration values")
	}
	return nil
}

func (c Config) WithProviderMetadata(ctx context.Context) (*Config, error) {
	idportenMetadata, err := oauth.Metadata(c.DigDir.IDPorten.WellKnownURL).OpenID(ctx)
	if err != nil {
		return nil, err
	}

	maskinportenMetadata, err := oauth.Metadata(c.DigDir.Maskinporten.WellKnownURL).OAuth(ctx)
	if err != nil {
		return nil, err
	}

	c.DigDir.IDPorten.Metadata = *idportenMetadata
	c.DigDir.Maskinporten.Metadata = *maskinportenMetadata

	return &c, nil
}

func decoderHook(dc *mapstructure.DecoderConfig) {
	dc.TagName = "json"
	dc.ErrorUnused = true
}

func New() (*Config, error) {
	var err error
	var cfg Config

	err = viper.ReadInConfig()
	if err != nil {
		if err.(viper.ConfigFileNotFoundError) != err {
			return nil, err
		}
	}

	flag.Parse()

	err = viper.BindPFlags(flag.CommandLine)
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&cfg, decoderHook)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
