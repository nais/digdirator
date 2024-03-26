package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/square/go-jose.v2"

	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/google"
)

const (
	ClientID = "client-id"
	JwkPath  = "jwk-path"
)

func init() {
	flag.String(ClientID, "", "The client ID for the DigDir client to update.")
	flag.String(JwkPath, "", "Path to the new public key in JWK format that should be added to the client.")
}

func main() {
	err := run()
	if err != nil {
		log.Errorf("run error: %+v", err)
		os.Exit(1)
	}

	log.Info("success.")
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := makeConfig(ctx)
	if err != nil {
		return err
	}

	client, err := makeClient(ctx, cfg)
	if err != nil {
		return err
	}

	// create new key set that contains new key
	desired, err := makeKeySet()
	if err != nil {
		return err
	}

	// fetch existing keys
	clientID := viper.GetString(ClientID)
	existing, err := client.GetKeys(ctx, clientID)
	if err != nil {
		return fmt.Errorf("fetching JWKS for client %q: %w", clientID, err)
	}

	seen := make(map[string]bool)
	for _, key := range desired.Keys {
		if !seen[key.KeyID] {
			seen[key.KeyID] = true
		}
	}

	// only add existing keys that do not have expired certificates to new key set
	for _, key := range existing.Keys {
		valid := true

		for _, cert := range key.Certificates {
			if time.Now().After(cert.NotAfter) {
				valid = false
				log.Warnf("found existing key %q with expired certificate (NotAfter %s); discarding...", key.KeyID, cert.NotAfter)
				break
			}
		}

		if valid && !seen[key.KeyID] {
			desired.Keys = append(desired.Keys, key)
		}
	}
	log.Infof("created new jwk key set: %s", keyIDs(desired))

	log.Infof("updating client %q with new key set...", clientID)
	response, err := client.RegisterKeys(ctx, clientID, desired)
	if err != nil {
		return fmt.Errorf("registering keys: %w", err)
	}

	log.Infof("update response: %s", keyIDs(&response.JSONWebKeySet))
	return nil
}

func makeConfig(ctx context.Context) (*config.Config, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("initializing config: %w", err)
	}

	cfg.Print([]string{})

	cfg, err = cfg.WithProviderMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching provider metadata: %w", err)
	}

	if err = cfg.Validate([]string{
		ClientID,
		config.DigDirAdminBaseURL,
		config.DigDirMaskinportenClientID,
		config.DigDirMaskinportenCertChain,
		config.DigDirMaskinportenScopes,
		config.DigDirMaskinportenWellKnownURL,
		config.DigDirMaskinportenKmsKeyPath,
	}); err != nil {
		return nil, err
	}

	return cfg, nil
}

func makeClient(ctx context.Context, cfg *config.Config) (digdir.Client, error) {
	sm, err := google.NewSecretManagerClient(ctx)
	if err != nil {
		return digdir.Client{}, fmt.Errorf("getting secret manager client: %v", err)
	}

	certs, err := sm.KeyChainMetadata(ctx, cfg.DigDir.Maskinporten.CertChain)

	if err != nil {
		return digdir.Client{}, fmt.Errorf("unable to fetch maskinporten cert chain: %w", err)
	}

	signer, err := crypto.NewKmsSigner(certs, cfg.DigDir.Maskinporten.KMS, ctx)
	if err != nil {
		return digdir.Client{}, fmt.Errorf("unable to setup signer: %w", err)
	}

	authClientID, err := sm.ClientIdMetadata(ctx, cfg.DigDir.Maskinporten.ClientID)

	if err != nil {
		return digdir.Client{}, fmt.Errorf("unable to fetch maskinporten client id: %w", err)
	}

	return digdir.NewClient(http.DefaultClient, signer, cfg, &naisiov1.MaskinportenClient{}, authClientID)
}

func makeKeySet() (*jose.JSONWebKeySet, error) {
	path := viper.GetString(JwkPath)
	set := &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{},
	}

	if len(path) == 0 {
		return set, nil
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("while reading jwk from %q: %w", path, err)
	}

	var jwk jose.JSONWebKey
	err = jwk.UnmarshalJSON(bytes)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling new jwk: %w", err)
	}

	log.Infof("loaded new key %q", jwk.KeyID)
	set.Keys = append(set.Keys, jwk)

	return set, nil
}

func keyIDs(keySet *jose.JSONWebKeySet) []string {
	res := make([]string, 0)
	for _, key := range keySet.Keys {
		res = append(res, key.KeyID)
	}
	return res
}
