package secrets

import (
	"context"
	"fmt"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/controllers/common/reconciler"
	"github.com/nais/digdirator/pkg/pods"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// +kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;delete;update;patch

type Client struct {
	ctx      context.Context
	instance common.Instance
	logger   *log.Entry
	reconciler.Reconciler
}

func NewClient(ctx context.Context, instance common.Instance, logger *log.Entry, reconciler reconciler.Reconciler) Client {
	return Client{
		ctx:        ctx,
		instance:   instance,
		logger:     logger,
		Reconciler: reconciler,
	}
}

func (s Client) CreateOrUpdate(jwk jose.JSONWebKey) error {
	s.logger.Infof("processing secret with name '%s'...", s.instance.SecretName())
	spec, err := OpaqueSecret(s.instance, jwk)
	if err != nil {
		return fmt.Errorf("creating secret spec: %w", err)
	}
	if err := ctrl.SetControllerReference(s.instance, spec, s.Scheme); err != nil {
		return fmt.Errorf("setting controller reference: %w", err)
	}

	err = s.Client.Create(s.ctx, spec)
	res := controllerutil.OperationResultCreated

	if errors.IsAlreadyExists(err) {
		err = s.Client.Update(s.ctx, spec)
		res = controllerutil.OperationResultUpdated
	}
	if err != nil {
		return fmt.Errorf("applying secretSpec: %w", err)
	}
	s.logger.Infof("secret '%s' %s", s.instance.SecretName(), res)
	return nil
}

func (s Client) GetManaged() (*Lists, error) {
	// fetch all application pods for this app
	podList, err := pods.GetForApplication(s.ctx, s.instance, s.Reader)
	if err != nil {
		return nil, err
	}

	// fetch all managed secrets
	var allSecrets corev1.SecretList
	opts := []client.ListOption{
		client.InNamespace(s.instance.GetNamespace()),
		client.MatchingLabels(s.instance.Labels()),
	}
	if err := s.Reader.List(s.ctx, &allSecrets, opts...); err != nil {
		return nil, err
	}

	// find intersect between secrets in use by application pods and all managed secrets
	podSecrets := podSecretLists(allSecrets, *podList)
	return &podSecrets, nil
}

func (s Client) DeleteUnused(unused corev1.SecretList) error {
	for _, oldSecret := range unused.Items {
		if oldSecret.Name == s.instance.SecretName() {
			continue
		}
		s.logger.Infof("deleting unused secret '%s'...", oldSecret.Name)
		if err := s.Delete(oldSecret); err != nil {
			return err
		}
	}
	return nil
}

func (s Client) Delete(secret corev1.Secret) error {
	if err := s.Client.Delete(s.ctx, &secret); err != nil {
		return fmt.Errorf("deleting unused secret: %w", err)
	}
	return nil
}
