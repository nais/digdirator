package common

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/nais/digdirator/pkg/digdir"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/scopes"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
)

type scope struct {
	*Reconciler
	Tx *Transaction
}

func (r *Reconciler) scopes(transaction *Transaction) scope {
	return scope{Reconciler: r, Tx: transaction}
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

func (s scope) createScopes(toCreate []naisiov1.ExposedScope) error {
	for _, newScope := range toCreate {
		s.Tx.Logger.Debug(fmt.Sprintf("Subscope %q does not exist in Digdir, creating...", newScope.Name))

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
		s.Tx.Logger.Debug(fmt.Sprintf("updating existing scope %q...", scope.ToString()))

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
	logger := s.Tx.Logger.WithField("scope", scopeName)

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
		logger.Info(msg)
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
		if consumer.ShouldBeAdded {
			if err := s.activateConsumer(scopeName, consumer.Orgno, logger); err != nil {
				if digdirErr, ok := asInvalidConsumerError(err); ok {
					logger.Warnf("ACL: consumer %q is invalid: %q; skipping...", consumer.Orgno, digdirErr.Message)
					invalidConsumers = append(invalidConsumers, consumer.Orgno)
					continue
				}
				return err
			}

			consumerStatus = append(consumerStatus, consumer.Orgno)
			metrics.IncScopesConsumersCreatedOrUpdated(s.Tx.Instance, consumer.State)
		} else {
			if err := s.deactivateConsumer(scopeName, consumer.Orgno, logger); err != nil {
				return err
			}

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

func (s scope) activateConsumer(scope, consumerOrgno string, logger *log.Entry) error {
	logger.Infof("ACL: adding consumer %q...", consumerOrgno)

	_, err := s.DigDirClient.AddToScopeACL(s.Tx.Ctx, scope, consumerOrgno)
	if err != nil {
		return fmt.Errorf("adding consumer: %w", err)
	}

	msg := fmt.Sprintf("ACL: granted access to scope %q for consumer %q", scope, consumerOrgno)
	logger.Info(msg)
	s.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, msg)

	return nil
}

func (s scope) deactivateConsumer(scope, consumerOrgno string, logger *log.Entry) error {
	logger.Infof("ACL: removing consumer %q...", consumerOrgno)

	_, err := s.DigDirClient.DeactivateConsumer(s.Tx.Ctx, scope, consumerOrgno)
	if err != nil {
		return fmt.Errorf("deactivating consumer: %w", err)
	}
	msg := fmt.Sprintf("ACL: revoked access to scope %q for consumer %q", scope, consumerOrgno)
	logger.Info(msg)
	s.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, msg)

	return nil
}

func (s scope) update(scope scopes.Scope) error {
	scopePayload := clients.ToScopeRegistration(s.Tx.Instance, scope.CurrentScope, s.Config)
	s.Tx.Logger.WithField("scope", scope.ToString()).Debug("updating scope...")

	registrationResponse, err := s.DigDirClient.UpdateScope(s.Tx.Ctx, scopePayload, scope.ToString())
	if err != nil {
		return fmt.Errorf("updating scope: %w", err)
	}

	msg := fmt.Sprintf("Updated scope %q", registrationResponse.Name)
	s.Tx.Logger.Info(msg)
	s.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedScopeInDigDir, msg)
	metrics.IncScopesUpdated(s.Tx.Instance)

	return nil
}

func (s scope) create(newScope naisiov1.ExposedScope) (*types.ScopeRegistration, error) {
	payload := clients.ToScopeRegistration(s.Tx.Instance, newScope, s.Config)
	s.Tx.Logger.Debug("scope does not exist in Digdir, registering...")

	response, err := s.DigDirClient.RegisterScope(s.Tx.Ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("registering scope: %w", err)
	}

	s.Tx.Logger.WithField("scope", response.Name).Info("scope registered")
	return response, nil
}

func (s scope) deactivate(scope scopes.Scope) error {
	registration, err := s.DigDirClient.DeleteScope(s.Tx.Ctx, scope.ToString())
	if err != nil {
		return fmt.Errorf("deleting scope: %w", err)
	}

	msg := fmt.Sprintf("Deactivated scope %q; consumers no longer have access", registration.Name)
	s.Tx.Logger.Warning(msg)
	s.reportEvent(s.Tx, corev1.EventTypeWarning, EventDeactivatedScopeInDigDir, msg)
	metrics.IncScopesDeleted(s.Tx.Instance)

	return nil
}

func (s scope) activate(scope scopes.Scope) error {
	payload := clients.ToScopeRegistration(s.Tx.Instance, scope.CurrentScope, s.Config)
	registration, err := s.DigDirClient.UpdateScope(s.Tx.Ctx, payload, scope.ToString())
	if err != nil {
		return fmt.Errorf("activating scope: %w", err)
	}

	msg := fmt.Sprintf("Activated scope %q", registration.Name)
	s.Tx.Logger.Info(msg)
	s.reportEvent(s.Tx, corev1.EventTypeNormal, EventActivatedScopeInDigDir, msg)
	metrics.IncScopesReactivated(s.Tx.Instance)

	return nil
}

func (s scope) Finalize(exposedScopes map[string]naisiov1.ExposedScope) error {
	filteredScopes, err := s.filtered(exposedScopes)
	if err != nil {
		return err
	}

	if filteredScopes == nil || len(filteredScopes.ToUpdate) == 0 {
		return nil
	}

	for _, scope := range filteredScopes.ToUpdate {
		if scope.CurrentScope.Enabled {
			s.Tx.Logger.Infof("scope %q is still set to enabled, skipping deletion... ", scope.ToString())
			continue
		}

		s.Tx.Logger.Infof("finalizer triggered, deleting scope %q from Maskinporten... ", scope.ToString())
		err := s.deactivate(scope)
		if err != nil {
			return err
		}
	}

	return nil
}
