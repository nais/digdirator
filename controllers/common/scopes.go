package common

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/nais/digdirator/pkg/digdir"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/scopes"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
)

type scope struct {
	*Reconciler
	Tx  *Transaction
	log logr.Logger
}

func (r *Reconciler) scopes(tx *Transaction) scope {
	return scope{
		Reconciler: r,
		Tx:         tx,
		log:        ctrl.LoggerFrom(tx.Ctx).WithValues("subsystem", "scopes"),
	}
}

func (s scope) Process(exposedScopes map[string]naisiov1.ExposedScope) error {
	if len(exposedScopes) == 0 {
		return nil
	}

	filtered, err := s.filtered(exposedScopes)
	if err != nil {
		return err
	}

	err = s.createScopes(filtered.ToCreate)
	if err != nil {
		return fmt.Errorf("creating scopes: %w", err)
	}

	err = s.updateScopes(filtered.ToUpdate)
	if err != nil {
		return fmt.Errorf("updating scopes: %w", err)
	}

	return nil
}

func (s scope) Finalize(exposedScopes map[string]naisiov1.ExposedScope) error {
	filtered, err := s.filtered(exposedScopes)
	if err != nil {
		return err
	}

	if filtered == nil || len(filtered.ToUpdate) == 0 {
		return nil
	}

	for _, scope := range filtered.ToUpdate {
		log := s.log.WithValues("scope", scope.ToString())
		if scope.CurrentScope.Enabled {
			log.Info(fmt.Sprintf("Scope %q is still set to enabled, skipping deletion... ", scope.ToString()))
			continue
		}

		log.Info(fmt.Sprintf("Finalizer triggered, deleting scope %q from Maskinporten... ", scope.ToString()))
		err := s.deactivate(scope)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s scope) createScopes(toCreate []naisiov1.ExposedScope) error {
	for _, newScope := range toCreate {
		s.log.V(4).Info(fmt.Sprintf("Subscope %q does not exist in Digdir, creating...", newScope.Name))

		scope, err := s.create(newScope)
		if err != nil {
			return err
		}
		s.reportEvent(s.Tx, corev1.EventTypeNormal, EventCreatedScopeInDigDir, fmt.Sprintf("Created scope %q", scope.Name))
		metrics.IncScopesCreated(s.Tx.Instance)

		// add consumers
		err = s.updateACL(scopes.CurrentScopeInfo(*scope, newScope))
		if err != nil {
			return fmt.Errorf("updating ACL: %w", err)
		}
	}

	return nil
}

func (s scope) updateScopes(toUpdate []scopes.Scope) error {
	for _, scope := range toUpdate {
		log := s.log.WithValues("scope", scope.ToString())
		log.V(4).Info(fmt.Sprintf("updating existing scope %q...", scope.ToString()))

		wantEnabled := scope.CurrentScope.Enabled
		isEnabled := scope.ScopeRegistration.Active

		var err error

		if wantEnabled && !isEnabled {
			err = s.activate(scope)
		} else if wantEnabled {
			err = s.update(scope)
		} else {
			err = s.deactivate(scope)
		}
		if err != nil {
			return err
		}

		if wantEnabled {
			err = s.updateACL(scope)
			if err != nil {
				return fmt.Errorf("updating ACL: %w", err)
			}
		}
	}

	return nil
}

func (s scope) filtered(exposedScopes map[string]naisiov1.ExposedScope) (*scopes.Operations, error) {
	allScopes, err := s.DigDirClient.GetScopes(s.Tx.Ctx)
	if err != nil {
		return nil, fmt.Errorf("getting scopes: %w", err)
	}

	return scopes.Generate(allScopes, exposedScopes), nil
}

func (s scope) updateACL(scope scopes.Scope) error {
	scopeName := scope.ToString()
	log := s.log.WithValues("scope", scopeName)

	acl, err := s.DigDirClient.GetScopeACL(s.Tx.Ctx, scopeName)
	if err != nil {
		return fmt.Errorf("getting ACL: %w", err)
	}

	consumerStatus, consumerList := scope.FilterConsumers(acl)
	setValidCondition := func() {
		s.Tx.Instance.GetStatus().SetCondition(InvalidExposedScopesConsumersCondition(
			metav1.ConditionFalse,
			ConditionReasonValidated,
			fmt.Sprintf("All consumers for scope %q are valid", scopeName),
			s.Tx.Instance.GetGeneration()),
		)
	}

	if len(consumerList) == 0 {
		msg := fmt.Sprintf("ACL: scope %q is up to date", scopeName)
		log.Info(msg)
		s.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, msg)
		setValidCondition()
		return nil
	}

	invalidConsumers := make([]string, 0)
	asInvalidConsumerError := func(err error) (*digdir.Error, bool) {
		var digdirError *digdir.Error
		if errors.As(err, &digdirError) && digdirError.StatusCode == http.StatusBadRequest {
			return digdirError, true
		}
		return nil, false
	}

	for _, consumer := range consumerList {
		log = s.log.WithValues("scope", scopeName, "consumer_orgno", consumer.Orgno)

		if consumer.ShouldBeAdded {
			log.Info(fmt.Sprintf("ACL: adding consumer %q...", consumer.Orgno))

			_, err := s.DigDirClient.AddToScopeACL(s.Tx.Ctx, scopeName, consumer.Orgno)
			if err != nil {
				if digdirErr, ok := asInvalidConsumerError(err); ok {
					log.Error(digdirErr, fmt.Sprintf("ACL: consumer %q is invalid; skipping...", consumer.Orgno))
					invalidConsumers = append(invalidConsumers, consumer.Orgno)
					continue
				}
				return fmt.Errorf("adding consumer: %w", err)
			}

			msg := fmt.Sprintf("ACL: granted access to scope %q for consumer %q", scopeName, consumer.Orgno)
			log.Info(msg)
			s.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, msg)

			consumerStatus = append(consumerStatus, consumer.Orgno)
			metrics.IncScopesConsumersCreatedOrUpdated(s.Tx.Instance, consumer.State)
		} else {
			log.Info(fmt.Sprintf("ACL: removing consumer %q...", consumer.Orgno))

			_, err := s.DigDirClient.DeactivateConsumer(s.Tx.Ctx, scopeName, consumer.Orgno)
			if err != nil {
				return fmt.Errorf("deactivating consumer: %w", err)
			}

			msg := fmt.Sprintf("ACL: revoked access to scope %q for consumer %q", scopeName, consumer.Orgno)
			log.Info(msg)
			s.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, msg)

			metrics.IncScopesConsumersDeleted(s.Tx.Instance)
		}
	}

	if len(invalidConsumers) > 0 {
		s.Tx.Instance.GetStatus().SetCondition(InvalidExposedScopesConsumersCondition(
			metav1.ConditionTrue,
			ConditionReasonFailed,
			fmt.Sprintf("Scope %q has invalid consumers [%s]", scopeName, strings.Join(invalidConsumers, ", ")),
			s.Tx.Instance.GetGeneration()),
		)
	} else {
		setValidCondition()
	}

	return nil
}

func (s scope) create(newScope naisiov1.ExposedScope) (*types.ScopeRegistration, error) {
	payload := clients.ToScopeRegistration(s.Tx.Instance, newScope, s.Config)
	response, err := s.DigDirClient.RegisterScope(s.Tx.Ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("registering scope: %w", err)
	}

	s.log.WithValues("scope", response.Name).Info("scope registered")
	return response, nil
}

func (s scope) update(scope scopes.Scope) error {
	payload := clients.ToScopeRegistration(s.Tx.Instance, scope.CurrentScope, s.Config)
	registration, err := s.DigDirClient.UpdateScope(s.Tx.Ctx, payload, scope.ToString())
	if err != nil {
		return fmt.Errorf("updating scope: %w", err)
	}

	msg := fmt.Sprintf("Updated scope %q", registration.Name)
	s.log.WithValues("scope", registration.Name).Info(msg)
	s.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedScopeInDigDir, msg)
	metrics.IncScopesUpdated(s.Tx.Instance)

	return nil
}

func (s scope) activate(scope scopes.Scope) error {
	payload := clients.ToScopeRegistration(s.Tx.Instance, scope.CurrentScope, s.Config)
	registration, err := s.DigDirClient.UpdateScope(s.Tx.Ctx, payload, scope.ToString())
	if err != nil {
		return fmt.Errorf("activating scope: %w", err)
	}

	msg := fmt.Sprintf("Activated scope %q", registration.Name)
	s.log.WithValues("scope", registration.Name).Info(msg)
	s.reportEvent(s.Tx, corev1.EventTypeNormal, EventActivatedScopeInDigDir, msg)
	metrics.IncScopesReactivated(s.Tx.Instance)

	return nil
}

func (s scope) deactivate(scope scopes.Scope) error {
	registration, err := s.DigDirClient.DeleteScope(s.Tx.Ctx, scope.ToString())
	if err != nil {
		return fmt.Errorf("deleting scope: %w", err)
	}

	msg := fmt.Sprintf("Deactivated scope %q; consumers no longer have access", registration.Name)
	s.log.WithValues("scope", registration.Name).Info(msg)
	s.reportEvent(s.Tx, corev1.EventTypeWarning, EventDeactivatedScopeInDigDir, msg)
	metrics.IncScopesDeleted(s.Tx.Instance)

	return nil
}
