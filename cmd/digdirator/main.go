package main

import (
	"fmt"
	"github.com/go-logr/zapr"
	"github.com/nais/digdirator/controllers/idportenclient"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/crypto"
	"github.com/nais/digdirator/pkg/idporten"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"

	naisiov1 "github.com/nais/digdirator/api/v1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = naisiov1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	err := run()

	if err != nil {
		setupLog.Error(err, "Run loop errored")
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

	jwk, err := crypto.LoadJwkFromPath(cfg.DigDir.Auth.JwkPath)
	if err != nil {
		return fmt.Errorf("loading jwk credentials: %v", err)
	}

	signer, err := crypto.SignerFromJwk(jwk)
	if err != nil {
		return fmt.Errorf("creating signer from jwk: %v", err)
	}

	if err = (&idportenclient.Reconciler{
		Client:         mgr.GetClient(),
		Reader:         mgr.GetAPIReader(),
		Scheme:         mgr.GetScheme(),
		Config:         cfg,
		Recorder:       mgr.GetEventRecorderFor("digdirator"),
		IDPortenClient: idporten.NewClient(signer, *cfg),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
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
	log.SetLevel(log.DebugLevel)

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
		config.DigDirAuthJwkPath,
		config.DigDirAuthScopes,
		config.DigDirAuthTokenEndpoint,
		config.DigDirIDPortenApiEndpoint,
	}); err != nil {
		return nil, err
	}
	return cfg, nil
}
