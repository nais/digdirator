package main

import (
	kms "cloud.google.com/go/kms/apiv1"
	"context"
	"fmt"
	"github.com/go-logr/zapr"
	"github.com/nais/digdirator/controllers/idportenclient"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/metrics"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
	"time"

	naisiov1 "github.com/nais/digdirator/api/v1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	ctrlMetrics.Registry.MustRegister(metrics.AllMetrics...)

	_ = clientgoscheme.AddToScheme(scheme)
	_ = naisiov1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme

	log.SetLevel(log.DebugLevel)
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
	cfg, err := setupConfig()
	if err != nil {
		return err
	}

	zapLogger, err := setupZapLogger()
	if err != nil {
		return err
	}
	ctrl.SetLogger(zapr.NewLogger(zapLogger))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: cfg.MetricsAddr,
		LeaderElection:     false,
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	signer, err := setupSigner(cfg)
	if err != nil {
		return fmt.Errorf("unable to setup signer: %w", err)
	}

	if err = (&idportenclient.Reconciler{
		Client:     mgr.GetClient(),
		Reader:     mgr.GetAPIReader(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("digdirator"),
		Config:     cfg,
		Signer:     signer,
		HttpClient: http.DefaultClient,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting metrics refresh goroutine")
	clusterMetrics := metrics.New(mgr.GetAPIReader())
	go clusterMetrics.Refresh(context.Background())

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}

func setupSigner(cfg *config.Config) (jose.Signer, error) {
	x5c, err := crypto.LoadPemCertChainToX5C(cfg.DigDir.Auth.CertChainPath)
	if err != nil {
		return nil, fmt.Errorf("loading PEM cert chain to X5C: %v", err)
	}
	kmsPath := crypto.KmsKeyPath(cfg.DigDir.Auth.KmsKeyPath)
	kmsCtx := context.Background()
	kmsClient, err := kms.NewKeyManagementClient(kmsCtx)
	if err != nil {
		return nil, fmt.Errorf("error creating key management client: %v", err)
	}
	return crypto.NewKmsSigner(&crypto.KmsOptions{
		Client:     kmsClient,
		Ctx:        kmsCtx,
		KmsKeyPath: kmsPath,
	}, x5c)
}

func setupZapLogger() (*zap.Logger, error) {
	if viper.GetBool(config.DevelopmentMode) {
		logger, err := zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
		return logger, nil
	}

	formatter := log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	log.SetFormatter(&formatter)

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	return loggerConfig.Build()
}

func setupConfig() (*config.Config, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, err
	}

	cfg.Print([]string{})

	if err = cfg.Validate([]string{
		config.ClusterName,
		config.DigDirAuthAudience,
		config.DigDirAuthClientID,
		config.DigDirAuthCertChainPath,
		config.DigDirAuthScopes,
		config.DigDirAuthBaseURL,
		config.DigDirIDPortenBaseURL,
		config.DigDirAuthKmsKeyPath,
	}); err != nil {
		return nil, err
	}
	return cfg, nil
}
