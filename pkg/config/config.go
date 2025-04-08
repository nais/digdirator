package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/liberator/pkg/oauth"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	MetricsAddr     string         `json:"metrics-address"`
	ClusterName     string         `json:"cluster-name"`
	DevelopmentMode bool           `json:"development-mode"`
	DigDir          DigDir         `json:"digdir"`
	Features        Features       `json:"features"`
	LeaderElection  LeaderElection `json:"leader-election"`
	LogLevel        string         `json:"log-level"`
}

type DigDir struct {
	Admin        Admin        `json:"admin"`
	IDPorten     IDPorten     `json:"idporten"`
	Maskinporten Maskinporten `json:"maskinporten"`
	Common       DigDirCommon `json:"common"`
}

type DigDirCommon struct {
	AccessTokenLifetime int    `json:"access-token-lifetime"`
	ClientName          string `json:"client-name"`
	ClientURI           string `json:"client-uri"`
	SessionLifetime     int    `json:"session-lifetime"`
}

type Admin struct {
	BaseURL    string `json:"base-url"`
	ClientID   string `json:"client-id"`
	CertChain  string `json:"cert-chain"`
	KMSKeyPath string `json:"kms-key-path"`
	Scopes     string `json:"scopes"`
}

type IDPorten struct {
	WellKnownURL string `json:"well-known-url"`
	Metadata     oauth.MetadataOpenID
}

type Maskinporten struct {
	Default           MaskinportenDefault `json:"default"`
	WellKnownURL      string              `json:"well-known-url"`
	Metadata          oauth.MetadataOAuth
	DelegationSources map[string]types.DelegationSource
}

type MaskinportenDefault struct {
	ClientScope string `json:"client-scope"`
	ScopePrefix string `json:"scope-prefix"`
}

type Features struct {
	Maskinporten bool `json:"maskinporten"`
}

type LeaderElection struct {
	Enabled   bool   `json:"enabled"`
	Namespace string `json:"namespace"`
}

const (
	LogLevel                = "log-level"
	MetricsAddress          = "metrics-address"
	ClusterName             = "cluster-name"
	DevelopmentMode         = "development-mode"
	LeaderElectionEnabled   = "leader-election.enabled"
	LeaderElectionNamespace = "leader-election.namespace"

	DigDirAdminBaseURL    = "digdir.admin.base-url"
	DigDirAdminClientID   = "digdir.admin.client-id"
	DigDirAdminCertChain  = "digdir.admin.cert-chain"
	DigDirAdminKmsKeyPath = "digdir.admin.kms-key-path"
	DigDirAdminScopes     = "digdir.admin.scopes"

	DigDirCommonClientName               = "digdir.common.client-name"
	DigDirCommonClientURI                = "digdir.common.client-uri"
	DigDirCommonAccessTokenLifetime      = "digdir.common.access-token-lifetime"
	DigDirCommonSessionLifetime          = "digdir.common.session-lifetime"
	DigDirIDPortenWellKnownURL           = "digdir.idporten.well-known-url"
	DigDirMaskinportenDefaultClientScope = "digdir.maskinporten.default.client-scope"
	DigDirMaskinportenDefaultScopePrefix = "digdir.maskinporten.default.scope-prefix"
	DigDirMaskinportenWellKnownURL       = "digdir.maskinporten.well-known-url"

	FeaturesMaskinporten = "features.maskinporten"
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
	flag.Bool(LeaderElectionEnabled, false, "Toggle for enabling leader election.")
	flag.String(LeaderElectionNamespace, "", "Namespace for the leader election resource. Needed if not running in-cluster (e.g. locally). If empty, will default to the same namespace as the running application.")
	flag.String(LogLevel, "info", "Log level for digdirator.")

	flag.String(DigDirAdminBaseURL, "", "Base URL endpoint for interacting with DigDir self service API")
	flag.String(DigDirAdminClientID, "", "Client ID / issuer for JWT assertion when authenticating with DigDir self service API.")
	flag.String(DigDirAdminScopes, "idporten:dcr.write idporten:dcr.read idporten:scopes.write", "List of space-separated scopes for JWT assertion when authenticating with DigDir self service API.")
	flag.String(DigDirAdminKmsKeyPath, "projects/<project-id>/locations/<location>/keyRings/<key-ring-name>/cryptoKeys/<key-name>/cryptoKeyVersions/<key-version>", "Resource path to Google KMS key used to sign JWT assertion.")
	flag.String(DigDirAdminCertChain, "", "Full certificate chain in PEM format for business certificate used to sign JWT assertion.")

	flag.String(DigDirCommonClientName, "ARBEIDS- OG VELFERDSETATEN", "Default name for all provisioned clients. Appears in the login prompt for ID-porten.")
	flag.String(DigDirCommonClientURI, "https://www.nav.no", "Default client URI for all provisioned clients. Appears in the back-button for the login prompt for ID-porten.")
	flag.Int(DigDirCommonAccessTokenLifetime, 3600, "Default lifetime (in seconds) for access tokens for all clients.")
	flag.Int(DigDirCommonSessionLifetime, 7200, "Default lifetime (in seconds) for sessions (authorization and refresh token lifetime) for all clients.")

	flag.String(DigDirIDPortenWellKnownURL, "", "URL to ID-porten well-known discovery metadata document.")
	flag.String(DigDirMaskinportenDefaultClientScope, "nav:test/api", "Default scope for provisioned Maskinporten clients, if none specified in spec.")
	flag.String(DigDirMaskinportenDefaultScopePrefix, "nav", "Default scope prefix for provisioned Maskinporten scopes.")
	flag.String(DigDirMaskinportenWellKnownURL, "", "URL to Maskinporten well-known discovery metadata document.")

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

	delegationSources, err := c.delegationSources(ctx)
	if err != nil {
		return nil, err
	}

	c.DigDir.IDPorten.Metadata = *idportenMetadata
	c.DigDir.Maskinporten.Metadata = *maskinportenMetadata
	c.DigDir.Maskinporten.DelegationSources = delegationSources

	return &c, nil
}

func (c Config) delegationSources(ctx context.Context) (map[string]types.DelegationSource, error) {
	delegationSourceURL, err := url.JoinPath(c.DigDir.Admin.BaseURL, "delegationsources")
	if err != nil {
		return nil, fmt.Errorf("parse delegation source url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, delegationSourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("make delegation source request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch delegation sources: %w", err)
	}
	defer resp.Body.Close()

	var delegationSources []types.DelegationSource
	err = json.NewDecoder(resp.Body).Decode(&delegationSources)
	if err != nil {
		return nil, fmt.Errorf("decode delegation sources: %w", err)
	}

	delegationSourceMap := make(map[string]types.DelegationSource)
	for _, source := range delegationSources {
		delegationSourceMap[source.Name] = source
	}

	return delegationSourceMap, nil
}

func decoderHook(dc *mapstructure.DecoderConfig) {
	dc.TagName = "json"
	dc.ErrorUnused = false
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
