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
	"github.com/nais/digdirator/internal/crypto/signer"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir"
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

var scheme = runtime.NewScheme()

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
}

func run() error {
	ctx := ctrl.SetupSignalHandler()
	cfg, err := setup(ctx)
	if err != nil {
		return err
	}

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
		return fmt.Errorf("starting manager: %w", err)
	}

	kmsSigner, err := signer.NewKmsSigner(ctx, cfg.DigDir.Admin.KMSKeyPath, []byte(cfg.DigDir.Admin.CertChain))
	if err != nil {
		return fmt.Errorf("setting up kms signer: %w", err)
	}

	digdirClient, err := digdir.NewClient(cfg, http.DefaultClient, kmsSigner)
	if err != nil {
		return fmt.Errorf("setting up digdir client: %w", err)
	}

	reconciler := common.NewReconciler(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		mgr.GetScheme(),
		mgr.GetEventRecorderFor("digdirator"),
		cfg,
		digdirClient,
	)

	if err = idportenclient.NewReconciler(reconciler).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("creating idportenclient controller: %w", err)
	}

	if cfg.Features.Maskinporten {
		if err = maskinportenclient.NewReconciler(reconciler).SetupWithManager(mgr); err != nil {
			return fmt.Errorf("creating maskinportenclient controller: %w", err)
		}
	}

	clusterMetrics := metrics.New(mgr.GetClient())
	go clusterMetrics.Refresh(ctx)

	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("running manager: %w", err)
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

	cfg.Print([]string{
		config.DigDirAdminCertChain,
	})

	required := []string{
		config.ClusterName,
		config.DigDirAdminBaseURL,
		config.DigDirAdminClientID,
		config.DigDirAdminCertChain,
		config.DigDirAdminKmsKeyPath,
		config.DigDirAdminScopes,
		config.DigDirIDPortenWellKnownURL,
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
