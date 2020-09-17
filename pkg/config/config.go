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
	MetricsAddr     string `json:"metrics-address"`
	ClusterName     string `json:"cluster-name"`
	DevelopmentMode bool   `json:"development-mode"`
	DigDir          DigDir `json:"digdir"`
}

type DigDir struct {
	Auth     Auth     `json:"auth"`
	IDPorten IDPorten `json:"idporten"`
}

type Auth struct {
	ClientID      string `json:"client-id"`
	CertChainPath string `json:"cert-chain-path"`
	BaseURL       string `json:"base-url"`
	Audience      string `json:"audience"`
	Scopes        string `json:"scopes"`
	KmsKeyPath    string `json:"kms-key-path"`
}

type IDPorten struct {
	BaseURL string `json:"base-url"`
}

const (
	MetricsAddress          = "metrics-address"
	ClusterName             = "cluster-name"
	DevelopmentMode         = "development-mode"
	DigDirAuthAudience      = "digdir.auth.audience"
	DigDirAuthClientID      = "digdir.auth.client-id"
	DigDirAuthCertChainPath = "digdir.auth.cert-chain-path"
	DigDirAuthScopes        = "digdir.auth.scopes"
	DigDirAuthBaseURL       = "digdir.auth.base-url"
	DigDirAuthKmsKeyPath    = "digdir.auth.kms-key-path"
	DigDirIDPortenBaseURL   = "digdir.idporten.base-url"
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
	flag.String(DigDirAuthAudience, "", "Audience for JWT assertion when authenticating to DigDir.")
	flag.String(DigDirAuthClientID, "", "Client ID / issuer for JWT assertion when authenticating to DigDir.")
	flag.String(DigDirAuthCertChainPath, "", "Path to PEM file containing certificate chain for authenticating to DigDir.")
	flag.String(DigDirAuthScopes, "", "List of scopes for JWT assertion when authenticating to DigDir.")
	flag.String(DigDirAuthBaseURL, "", "Base URL endpoint for authenticating to DigDir.")
	flag.String(DigDirIDPortenBaseURL, "", "Base URL endpoint for interacting with IDPorten API.")
	flag.String(DigDirAuthKmsKeyPath, "", "Full path to key including version in Google Cloud KMS.")
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
