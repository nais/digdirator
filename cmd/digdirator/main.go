package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/zapr"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/controllers/idportenclient"
	"github.com/nais/digdirator/controllers/maskinportenclient"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/google"
	"github.com/nais/digdirator/pkg/metrics"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	ctrlmetricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	ctrlmetrics.Registry.MustRegister(metrics.AllMetrics...)

	_ = clientgoscheme.AddToScheme(scheme)
	_ = nais_io_v1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	err := run()

	if err != nil {
		log.Error(err, "Run loop errored")
		os.Exit(1)
	}

	setupLog.Info("Manager shutting down")
}

func run() error {
	ctx := ctrl.SetupSignalHandler()
	cfg, err := setup(ctx)
	if err != nil {
		return err
	}

	setupLog.Info("instantiating manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: ctrlmetricsserver.Options{
			BindAddress: cfg.MetricsAddr,
		},
		LeaderElection:          cfg.LeaderElection.Enabled,
		LeaderElectionID:        "digdirator.nais.io",
		LeaderElectionNamespace: cfg.LeaderElection.Namespace,
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	setupLog.Info("instantiating secret manager client")
	secretManagerClient, err := google.NewSecretManagerClient(ctx)
	if err != nil {
		return fmt.Errorf("getting secret manager client: %v", err)
	}

	setupLog.Info("fetching certificate key chain for idporten")
	idportenKeyChain, err := secretManagerClient.KeyChainMetadata(
		ctx,
		cfg.DigDir.IDPorten.CertificateChain,
	)

	if err != nil {
		return fmt.Errorf("unable to fetch idporten cert chain: %w", err)
	}

	setupLog.Info("setting up signer for idporten")
	idportenSigner, err := crypto.NewKmsSigner(
		idportenKeyChain,
		cfg.DigDir.IDPorten.KMS,
		ctx,
	)
	if err != nil {
		return fmt.Errorf("unable to setup signer: %w", err)
	}

	setupLog.Info("fetching client-id for idporten")
	idportenClientId, err := secretManagerClient.ClientIdMetadata(
		ctx,
		cfg.DigDir.IDPorten.ClientID,
	)

	if err != nil {
		return fmt.Errorf("unable to fetch idporten client id: %w", err)
	}

	setupLog.Info("instantiating reconciler for idporten")
	idportenReconciler := idportenclient.NewReconciler(common.NewReconciler(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("digdirator"),
		cfg,
		idportenSigner,
		http.DefaultClient,
		idportenClientId,
	))

	setupLog.Info("setting up idporten reconciler with manager")
	if err = idportenReconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}
	// +kubebuilder:scaffold:builder

	if cfg.Features.Maskinporten {
		setupLog.Info("fetching certificate key chain for maskinporten")
		maskinportenKeyChain, err := secretManagerClient.KeyChainMetadata(
			ctx,
			cfg.DigDir.Maskinporten.CertChain,
		)

		if err != nil {
			return fmt.Errorf("unable to fetch maskinporten cert chain: %w", err)
		}

		setupLog.Info("setting up signer for maskinporten")
		maskinportenSigner, err := crypto.NewKmsSigner(
			maskinportenKeyChain,
			cfg.DigDir.Maskinporten.KMS,
			ctx,
		)
		if err != nil {
			return fmt.Errorf("unable to setup signer: %w", err)
		}

		setupLog.Info("fetching client-id for maskinporten")
		maskinportenClientId, err := secretManagerClient.ClientIdMetadata(
			ctx,
			cfg.DigDir.Maskinporten.ClientID,
		)

		if err != nil {
			return fmt.Errorf("unable to fetch maskinporten client id: %w", err)
		}

		setupLog.Info("instantiating reconciler for maskinporten")
		maskinportenReconciler := maskinportenclient.NewReconciler(
			common.NewReconciler(
				mgr.GetClient(),
				mgr.GetAPIReader(),
				mgr.GetScheme(),
				mgr.GetEventRecorderFor("digdirator"),
				cfg,
				maskinportenSigner,
				http.DefaultClient,
				maskinportenClientId,
			))

		setupLog.Info("setting up maskinporten reconciler with manager")
		if err = maskinportenReconciler.SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create controller: %w", err)
		}
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting metrics refresh goroutine")
	clusterMetrics := metrics.New(mgr.GetClient())
	go clusterMetrics.Refresh(ctx)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}

func setup(ctx context.Context) (*config.Config, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}

	if err := setupLoggers(cfg.LogLevel, cfg.DevelopmentMode); err != nil {
		return nil, err
	}

	cfg.Print([]string{})

	required := []string{
		config.ClusterName,
		config.DigDirAdminBaseURL,
		config.DigDirIDportenClientID,
		config.DigDirIDportenCertChain,
		config.DigDirIDportenKmsKeyPath,
		config.DigDirIDportenScopes,
		config.DigDirIDPortenWellKnownURL,
		config.DigDirMaskinportenClientID,
		config.DigDirMaskinportenCertChain,
		config.DigDirMaskinportenKmsKeyPath,
		config.DigDirMaskinportenScopes,
		config.DigDirMaskinportenWellKnownURL,
	}

	if err = cfg.Validate(required); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	return cfg.WithProviderMetadata(ctx)
}

func setupLoggers(logLevel string, developmentMode bool) error {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	log.SetLevel(func() log.Level {
		level, err := log.ParseLevel(logLevel)
		if err != nil {
			return log.InfoLevel
		}
		return level
	}())

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	cfg.Level = func() zap.AtomicLevel {
		level, err := zap.ParseAtomicLevel(logLevel)
		if err != nil {
			return zap.NewAtomicLevelAt(zapcore.InfoLevel)
		}
		return level
	}()

	logger, err := cfg.Build()
	if err != nil {
		return err
	}

	if developmentMode {
		logger, err = zap.NewDevelopment()
		if err != nil {
			return err
		}
	}

	ctrl.SetLogger(zapr.NewLogger(logger))
	return nil
}
