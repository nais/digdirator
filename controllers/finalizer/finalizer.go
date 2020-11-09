package finalizer

import (
	"context"
	"fmt"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/pkg/digdir"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const FinalizerName string = "finalizer.digdirator.nais.io"

// Finalizers allow the controller to implement an asynchronous pre-delete hook

type Finalizer struct {
	client       client.Client
	ctx          context.Context
	logger       *log.Entry
	recorder     record.EventRecorder
	digdirClient digdir.Client
	clientID     string
}

func NewFinalizer(
	client client.Client,
	ctx context.Context,
	logger *log.Entry,
	recorder record.EventRecorder,
	digdirClient digdir.Client,
	clientID string,
) Finalizer {
	return Finalizer{
		client:       client,
		ctx:          ctx,
		logger:       logger,
		recorder:     recorder,
		digdirClient: digdirClient,
		clientID:     clientID,
	}
}

func (f Finalizer) Register(instance common.Instance) (ctrl.Result, error) {
	if !common.HasFinalizer(instance, FinalizerName) {
		f.logger.Info("finalizer for object not found, registering...")
		controllerutil.AddFinalizer(instance, FinalizerName)
		if err := f.client.Update(f.ctx, instance); err != nil {
			return ctrl.Result{}, fmt.Errorf("registering finalizer: %w", err)
		}
		f.recorder.Event(instance, corev1.EventTypeNormal, "Added", "Object finalizer is added")
	}
	return ctrl.Result{}, nil
}

func (f Finalizer) Process(instance common.Instance) (ctrl.Result, error) {
	if !common.HasFinalizer(instance, FinalizerName) {
		return ctrl.Result{}, nil
	}

	f.logger.Info("finalizer triggered, deleting resources...")

	if len(f.clientID) == 0 {
		return ctrl.Result{}, nil
	}
	if err := f.client.Delete(f.ctx, instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("deleting client from ID-porten: %w", err)
	}

	controllerutil.RemoveFinalizer(instance, FinalizerName)
	if err := f.client.Update(f.ctx, instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("removing finalizer from list: %w", err)
	}

	f.logger.Info("finalizer processing completed")
	return ctrl.Result{}, nil
}
