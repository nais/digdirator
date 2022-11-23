package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/go-logr/zapr"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"

	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/controllers/idportenclient"
	"github.com/nais/digdirator/controllers/maskinportenclient"
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
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/nais/digdirator/pkg/google"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	ctrlMetrics.Registry.MustRegister(metrics.AllMetrics...)

	_ = clientgoscheme.AddToScheme(scheme)
	_ = nais_io_v1.AddToScheme(scheme)
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

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	cfg, err = cfg.WithProviderMetadata(ctx)
	if err != nil {
		return err
	}

	zapLogger, err := setupZapLogger()
	if err != nil {
		return err
	}
	ctrl.SetLogger(zapr.NewLogger(zapLogger))

	setupLog.Info("instantiating manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: cfg.MetricsAddr,
		LeaderElection:     false,
	})
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	ctx = context.Background()

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
	idportenSigner, err := setupSigner(
		idportenKeyChain,
		cfg.DigDir.IDPorten.KMS,
		ctx,
	)
	if err != nil {
		return fmt.Errorf("unable to setup signer: %w", err)
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
		maskinportenSigner, err := setupSigner(
			maskinportenKeyChain,
			cfg.DigDir.Maskinporten.KMS,
			ctx,
		)
		if err != nil {
			return fmt.Errorf("unable to setup signer: %w", err)
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
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}

func setupSigner(certChain []byte, kmsConfig config.KMS, ctx context.Context) (jose.Signer, error) {
	signerOpts, err := crypto.SetupSignerOptions(certChain)
	if err != nil {
		return nil, fmt.Errorf("setting up signer options: %v", err)
	}

	kmsCtx := ctx
	kmsClient, err := kms.NewKeyManagementClient(kmsCtx)
	if err != nil {
		return nil, fmt.Errorf("error creating key management client: %v", err)
	}
	return crypto.NewKmsSigner(&crypto.KmsOptions{
		Client:    kmsClient,
		Ctx:       kmsCtx,
		KmsConfig: kmsConfig,
	}, signerOpts)
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
		config.DigDirAdminBaseURL,
		config.DigDirIDportenClientID,
		config.DigDirMaskinportenClientID,
		config.DigDirIDportenCertChainSecretName,
		config.DigDirMaskinportenCertChainSecretName,
		config.DigDirMaskinportenCertChainSecretProjectID,
		config.DigDirMaskinportenCertChainSecretVersion,
		config.DigDirIDportenCertChainSecretProjectID,
		config.DigDirIDportenCertChainSecretVersion,
		config.DigDirIDportenScopes,
		config.DigDirMaskinportenScopes,
		config.DigDirIDPortenWellKnownURL,
		config.DigDirMaskinportenWellKnownURL,
		config.DigDirIDportenKmsKey,
		config.DigDirMaskinportenKmsKey,
	}); err != nil {
		return nil, err
	}
	return cfg, nil
}
