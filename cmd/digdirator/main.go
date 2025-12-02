package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/controllers/idportenclient"
	"github.com/nais/digdirator/controllers/maskinportenclient"
	"github.com/nais/digdirator/internal/crypto/signer"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/metrics"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
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
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctrlmetrics.Registry.MustRegister(metrics.AllMetrics...)

	_ = clientgoscheme.AddToScheme(scheme)
	_ = nais_io_v1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	err := run()
	if err != nil {
		slog.Error("Run loop errored", "error", err)
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

	if err := setupLogger(cfg.LogLevel); err != nil {
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

func setupLogger(logLevel string) error {
	var level slog.Level
	if err := level.UnmarshalText([]byte(logLevel)); err != nil {
		return fmt.Errorf("parsing log level: %w", err)
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	slog.SetDefault(slog.New(handler))
	ctrl.SetLogger(logr.FromSlogHandler(handler))

	return nil
}
