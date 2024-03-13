package common

import (
	"fmt"

	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/scopes"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
)

type scope struct {
	Rec Reconciler
	Tx  *Transaction
}

func (r Reconciler) scopes(transaction *Transaction) scope {
	return scope{Rec: r, Tx: transaction}
}

func (s *scope) Process(exposedScopes map[string]naisiov1.ExposedScope) error {
	if exposedScopes == nil || len(exposedScopes) == 0 {
		return nil
	}

	filtered, err := s.filtered(exposedScopes)
	if err != nil {
		return fmt.Errorf("checking if scopes exists: %w", err)
	}

	err = s.createScopes(filtered.ToCreate)
	if err != nil {
		return err
	}

	err = s.updateScopes(filtered.ToUpdate)
	if err != nil {
		return err
	}

	return nil
}

func (s *scope) createScopes(toCreate []naisiov1.ExposedScope) error {
	for _, newScope := range toCreate {
		s.Tx.Logger.Debug(fmt.Sprintf("Subscope %q does not exist in Digdir, creating...", newScope.Name))

		scope, err := s.create(newScope)
		if err != nil {
			return err
		}
		s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventCreatedScopeInDigDir, fmt.Sprintf("Created scope %q", scope.Name))
		metrics.IncScopesCreated(s.Tx.Instance)

		// add consumers
		err = s.updateConsumers(scopes.CurrentScopeInfo(*scope, newScope))
		if err != nil {
			return fmt.Errorf("adding new consumers to acl: %w", err)
		}
	}

	return nil
}

func (s *scope) updateScopes(toUpdate []scopes.Scope) error {
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

		err = s.updateConsumers(scope)
		if err != nil {
			return fmt.Errorf("updating scope acl: %w", err)
		}
	}

	return nil
}

func (s *scope) filtered(exposedScopes map[string]naisiov1.ExposedScope) (*scopes.Operations, error) {
	allScopes, err := s.Tx.DigdirClient.GetScopes(s.Tx.Ctx)
	if err != nil {
		return nil, fmt.Errorf("getting filtered scopes: %w", err)
	}

	return scopes.Generate(allScopes, exposedScopes), nil
}

func (s *scope) updateConsumers(scope scopes.Scope) error {
	logger := s.Tx.Logger.WithField("scope", scope.ToString())
	logger.Debug("checking if ACL needs update...")

	acl, err := s.Tx.DigdirClient.GetScopeACL(s.Tx.Ctx, scope.ToString())
	if err != nil {
		return fmt.Errorf("getting ACL from Digdir: %w", err)
	}

	consumerStatus, consumerList := scope.FilterConsumers(acl)

	if len(consumerList) == 0 {
		msg := fmt.Sprintf("ACL was up to date for scope %q", scope.ToString())
		logger.Info(msg)
		s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, msg)
		return nil
	}

	for _, consumer := range consumerList {
		logger.Debug("processing ACLs...")
		if consumer.ShouldBeAdded {
			logger.Infof("adding consumer %q to ACL...", consumer.Orgno)
			err := s.activateConsumer(scope.ToString(), consumer.Orgno)
			if err != nil {
				return fmt.Errorf("adding consumer to ACL: %w", err)
			}

			consumerStatus = append(consumerStatus, consumer.Orgno)
			metrics.IncScopesConsumersCreatedOrUpdated(s.Tx.Instance, consumer.State)
		} else {
			logger.Infof("removing consumer %q from ACL...", consumer.Orgno)
			err := s.deactivateConsumer(scope.ToString(), consumer.Orgno)
			if err != nil {
				return fmt.Errorf("deleting consumer from ACL: %w", err)
			}

			metrics.IncScopesConsumersDeleted(s.Tx.Instance)
		}
	}

	return nil
}

func (s *scope) activateConsumer(scope, consumerOrgno string) error {
	response, err := s.Tx.DigdirClient.AddToScopeACL(s.Tx.Ctx, scope, consumerOrgno)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("Granted access to scope %q for consumer %q", scope, consumerOrgno)
	s.Tx.Logger.WithField("scope", response.Scope).Info(msg)
	s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, msg)

	return nil
}

func (s *scope) deactivateConsumer(scope, consumerOrgno string) error {
	response, err := s.Tx.DigdirClient.DeactivateConsumer(s.Tx.Ctx, scope, consumerOrgno)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("Revoked access to scope %q for consumer %q", scope, consumerOrgno)
	s.Tx.Logger.WithField("scope", response.Scope).Info(msg)
	s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, msg)

	return nil
}

func (s *scope) update(scope scopes.Scope) error {
	scopePayload := clients.ToScopeRegistration(s.Tx.Instance, scope.CurrentScope, s.Tx.Config)
	s.Tx.Logger.WithField("scope", scope.ToString()).Debug("updating scope...")

	registrationResponse, err := s.Tx.DigdirClient.UpdateScope(s.Tx.Ctx, scopePayload, scope.ToString())
	if err != nil {
		return fmt.Errorf("updating scope at Digdir: %w", err)
	}

	msg := fmt.Sprintf("Updated scope %q", registrationResponse.Name)
	s.Tx.Logger.Info(msg)
	s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedScopeInDigDir, msg)
	metrics.IncScopesUpdated(s.Tx.Instance)

	return nil
}

func (s *scope) create(newScope naisiov1.ExposedScope) (*types.ScopeRegistration, error) {
	payload := clients.ToScopeRegistration(s.Tx.Instance, newScope, s.Tx.Config)
	s.Tx.Logger.Debug("scope does not exist in Digdir, registering...")

	response, err := s.Tx.DigdirClient.RegisterScope(s.Tx.Ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("registering client to Digdir: %w", err)
	}

	s.Tx.Logger.WithField("scope", response.Name).Info("scope registered")
	return response, nil
}

func (s *scope) deactivate(scope scopes.Scope) error {
	registration, err := s.Tx.DigdirClient.DeleteScope(s.Tx.Ctx, scope.ToString())
	if err != nil {
		return fmt.Errorf("deleting scope: %w", err)
	}

	msg := fmt.Sprintf("Deactivated scope %q; consumers no longer have access", registration.Name)
	s.Tx.Logger.Warning(msg)
	s.Rec.reportEvent(s.Tx, corev1.EventTypeWarning, EventDeactivatedScopeInDigDir, msg)
	metrics.IncScopesDeleted(s.Tx.Instance)

	return nil
}

func (s *scope) activate(scope scopes.Scope) error {
	payload := clients.ToScopeRegistration(s.Tx.Instance, scope.CurrentScope, s.Tx.Config)
	registration, err := s.Tx.DigdirClient.UpdateScope(s.Tx.Ctx, payload, scope.ToString())
	if err != nil {
		return fmt.Errorf("activating scope: %w", err)
	}

	msg := fmt.Sprintf("Activated scope %q", registration.Name)
	s.Tx.Logger.Info(msg)
	s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventActivatedScopeInDigDir, msg)
	metrics.IncScopesReactivated(s.Tx.Instance)

	return nil
}

func (s *scope) Finalize(exposedScopes map[string]naisiov1.ExposedScope) error {
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
		if _, err := s.Tx.DigdirClient.DeleteScope(s.Tx.Ctx, scope.ToString()); err != nil {
			return fmt.Errorf("deactivate scope in Maskinporten: %w", err)
		}
		metrics.IncScopesDeleted(s.Tx.Instance)
		s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventDeactivatedScopeInDigDir, fmt.Sprintf("Deactivated scope %q", scope.ToString()))
	}

	return nil
}
