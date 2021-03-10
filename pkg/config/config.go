package config

import (
	"errors"
	"sort"
	"strings"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	MetricsAddr     string   `json:"metrics-address"`
	ClusterName     string   `json:"cluster-name"`
	ProjectID       string   `json:"project-id"`
	DevelopmentMode bool     `json:"development-mode"`
	DigDir          DigDir   `json:"digdir"`
	Features        Features `json:"features"`
}

type DigDir struct {
	Admin        Admin        `json:"admin"`
	Auth         Auth         `json:"auth"`
	IDPorten     IDPorten     `json:"idporten"`
	Maskinporten Maskinporten `json:"maskinporten"`
}

type Admin struct {
	BaseURL string `json:"base-url"`
}

type Auth struct {
	Audience string `json:"audience"`
	Scopes   string `json:"scopes"`
}

type IDPorten struct {
	BaseURL             string `json:"base-url"`
	KmsKeyPath          string `json:"kms-key-path"`
	ClientID            string `json:"client-id"`
	CertChainSecretName string `json:"cert-chain-secret-name"`
}

type Maskinporten struct {
	BaseURL             string `json:"base-url"`
	KmsKeyPath          string `json:"kms-key-path"`
	ClientID            string `json:"client-id"`
	CertChainSecretName string `json:"cert-chain-secret-name"`
}

type Features struct {
	Maskinporten bool `json:"maskinporten"`
}

const (
	MetricsAddress                        = "metrics-address"
	ClusterName                           = "cluster-name"
	ProjectID                             = "project-id"
	DevelopmentMode                       = "development-mode"
	DigDirAdminBaseURL                    = "digdir.admin.base-url"
	DigDirAuthAudience                    = "digdir.auth.audience"
	DigDirIDportenClientID                = "digdir.idporten.client-id"
	DigDirIDportenCertChainSecretName     = "digdir.idporten.cert-chain-secret-name"
	DigDirMaskinportenCertChainSecretName = "digdir.maskinporten.cert-chain-secret-name"
	DigDirMaskinportenClientID            = "digdir.maskinporten.client-id"
	DigDirAuthScopes                      = "digdir.auth.scopes"
	DigDirIDportenKmsKeyPath              = "digdir.idporten.kms-key-path"
	DigDirMaskinportenKmsKeyPath          = "digdir.maskinporten.kms-key-path"
	DigDirIDPortenBaseURL                 = "digdir.idporten.base-url"
	DigDirMaskinportenBaseURL             = "digdir.maskinporten.base-url"
	FeaturesMaskinporten                  = "features.maskinporten"
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
	flag.String(ProjectID, "", "The gcp project to fetch cert chain secret for the cluster.")
	flag.String(DevelopmentMode, "false", "Toggle for development mode.")
	flag.String(DigDirAdminBaseURL, "", "Base URL endpoint for interacting with Digdir Client Registration API")
	flag.String(DigDirAuthAudience, "", "Audience for JWT assertion when authenticating to DigDir.")
	flag.String(DigDirIDportenClientID, "", "Client ID / issuer for JWT assertion when authenticating to DigDir.")
	flag.String(DigDirIDportenKmsKeyPath, "", "Full path to key including version in Google Cloud KMS.")
	flag.String(DigDirMaskinportenClientID, "", "Client ID / issuer for JWT assertion when authenticating to DigDir.")
	flag.String(DigDirIDportenCertChainSecretName, "", "Secret name to PEM file containing certificate chain for authenticating to DigDir.")
	flag.String(DigDirMaskinportenCertChainSecretName, "", "Secret name to PEM file containing certificate chain for authenticating to DigDir.")
	flag.String(DigDirMaskinportenKmsKeyPath, "", "Full path to key including version in Google Cloud KMS.")
	flag.String(DigDirAuthScopes, "", "List of scopes for JWT assertion when authenticating to DigDir.")
	flag.String(DigDirMaskinportenBaseURL, "", "Base URL endpoint for interacting with Maskinporten API.")
	flag.String(DigDirIDPortenBaseURL, "", "Base URL endpoint for interacting with IDPorten API.")
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
