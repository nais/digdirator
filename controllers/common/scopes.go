package common

import (
	"fmt"
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir/scopes"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/metrics"
	naisiov1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	corev1 "k8s.io/api/core/v1"
)

type scope struct {
	Rec Reconciler
	Tx  *Transaction
}

func (r Reconciler) scopes(transaction *Transaction) scope {
	return scope{Rec: r, Tx: transaction}
}

func (s *scope) process(exposedScopes map[string]naisiov1.ExposedScope) error {
	filteredScopes, err := s.scopesExist(exposedScopes)

	if err != nil {
		return fmt.Errorf("checking if scopes exists: %w", err)
	}

	if len(filteredScopes.Current) > 0 {
		for _, scope := range filteredScopes.Current {
			s.Tx.Logger.Debug(fmt.Sprintf("Scope - %s already exists in Digdir...", scope.ToString()))

			if scope.HasChanged() {
				// update existing scope
				scopeRegistration, err := s.update(scope)
				if err != nil {
					return err
				}
				s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedScopeInDigDir, fmt.Sprintf("Scope updated.. %s", scopeRegistration.Name))
				metrics.IncScopesUpdated(s.Tx.Instance)
			}

			if scope.CanBeActivated() {
				// re-activate scope
				scopeRegistration, err := s.activate(scope)
				if err != nil {
					return err
				}
				s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventActivatedScopeInDigDir, fmt.Sprintf("Scope activated.. %s", scopeRegistration.Name))
				metrics.IncScopesReactivated(s.Tx.Instance)
			}

			_, err := s.updateConsumers(scope)
			if err != nil {
				return fmt.Errorf("update consumers acl: %w", err)
			}

			if !scope.IsActive() {
				// delete/deactivate scope
				scopeRegistration, err := s.deactivate(scope.ToString())
				if err != nil {
					return err
				}
				msg := fmt.Sprintf("Scope deactivated and no consumers is granted access... %s", scopeRegistration.Name)
				s.Tx.Logger.Warning(msg)
				s.Rec.reportEvent(s.Tx, corev1.EventTypeWarning, EventDeactivatedScopeInDigDir, msg)
				metrics.IncScopesDeleted(s.Tx.Instance)
			}
		}
	}

	if len(filteredScopes.ToCreate) > 0 {
		for _, newScope := range filteredScopes.ToCreate {
			s.Tx.Logger.Debug(fmt.Sprintf("Subscope - %s do not exist in Digdir, creating...", newScope.Name))

			scope, err := s.create(newScope)
			if err != nil {
				return err
			}
			s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventCreatedScopeInDigDir, fmt.Sprintf("Scope created.. %s", scope.Name))
			metrics.IncScopesCreated(s.Tx.Instance)
			// add consumers
			_, err = s.updateConsumers(scopes.CurrentScopeInfo(*scope, newScope))
			if err != nil {
				return fmt.Errorf("adding new consumers to acl: %w", err)
			}
		}
	}
	return nil
}

func (s scope) scopesExist(exposedScopes map[string]naisiov1.ExposedScope) (*scopes.ScopeStash, error) {
	scopeContainer, err := s.Tx.DigdirClient.GetFilteredScopes(s.Tx.Instance, s.Tx.Ctx, exposedScopes)
	if err != nil {
		return nil, fmt.Errorf("getting filterted scopes: %w", err)
	}
	return scopeContainer, nil
}

func (s *scope) updateConsumers(scope scopes.Scope) ([]types.ConsumerRegistration, error) {
	s.Tx.Logger = s.Tx.Logger.WithField("scope", scope.ToString())
	s.Tx.Logger.Debug("checking if ACL needs update...")

	acl, err := s.Tx.DigdirClient.GetScopeACL(s.Tx.Ctx, scope.ToString())
	if err != nil {
		return nil, fmt.Errorf("gettin ACL from Digdir: %w", err)
	}

	consumerStatus, consumerList := scope.FilterConsumers(acl)
	registrationResponse := make([]types.ConsumerRegistration, 0)

	if len(consumerList) == 0 {
		s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, fmt.Sprintf("ACL was up to date for: %s", scope.ToString()))
		return nil, nil
	}

	for _, consumer := range consumerList {
		if consumer.ShouldBeAdded {
			response, err := s.activateConsumer(scope.ToString(), consumer.Orgno)
			if err != nil {
				return nil, fmt.Errorf("adding to ACL: %w", err)
			}
			consumerStatus = append(consumerStatus, consumer.Orgno)
			registrationResponse = append(registrationResponse, *response)
			metrics.IncScopesConsumersCreatedOrUpdated(s.Tx.Instance, consumer.State)
		} else {
			response, err := s.deactivateConsumer(scope.ToString(), consumer.Orgno)
			if err != nil {
				return nil, fmt.Errorf("delete from ACL: %w", err)
			}
			registrationResponse = append(registrationResponse, *response)
			metrics.IncScopesConsumersDeleted(s.Tx.Instance)
		}
	}

	s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventUpdatedACLForScopeInDigDir, fmt.Sprintf("Scope ACL been updated.. %s", scope.ToString()))
	return registrationResponse, nil
}

func (s *scope) activateConsumer(scope, consumerOrgno string) (*types.ConsumerRegistration, error) {
	response, err := s.Tx.DigdirClient.AddToScopeACL(s.Tx.Ctx, scope, consumerOrgno)
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("scope acl updated, added consumer: %s", consumerOrgno)
	s.Tx.Logger.WithField("activateConsumer", response.Scope).Info(msg)
	return response, nil
}

func (s *scope) deactivateConsumer(scope, consumerOrgno string) (*types.ConsumerRegistration, error) {
	response, err := s.Tx.DigdirClient.DeactivateConsumer(s.Tx.Ctx, scope, consumerOrgno)
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("scope acl updated, deactivated consumer: %s", consumerOrgno)
	s.Tx.Logger.WithField("scope", response.Scope).Info(msg)
	return response, nil
}

func (s *scope) update(scope scopes.Scope) (*types.ScopeRegistration, error) {
	scopePayload := clients.ToScopeRegistration(s.Tx.Instance, scope.CurrentScope)
	s.Tx.Logger = s.Tx.Logger.WithField("scope", scope.ToString())
	s.Tx.Logger.Debug("updating scope...")

	registrationResponse, err := s.Tx.DigdirClient.UpdateScope(s.Tx.Ctx, scopePayload, scope.ToString())
	if err != nil {
		return nil, fmt.Errorf("updating scope at Digdir: %w", err)
	}
	return registrationResponse, err
}

func (s *scope) create(newScope naisiov1.ExposedScope) (*types.ScopeRegistration, error) {
	scopeRegistrationPayload := clients.ToScopeRegistration(s.Tx.Instance, newScope)
	s.Tx.Logger.Debug("scope does not exist in Digdir, registering...")

	registrationResponse, err := s.Tx.DigdirClient.RegisterScope(s.Tx.Ctx, scopeRegistrationPayload)
	if err != nil {
		return nil, fmt.Errorf("registering client to Digdir: %w", err)
	}

	s.Tx.Logger = s.Tx.Logger.WithField("scope", registrationResponse.Name)
	s.Tx.Logger.Info("scope registered")
	return registrationResponse, nil
}

func (s *scope) deactivate(scope string) (*types.ScopeRegistration, error) {
	scopeRegistration, err := s.Tx.DigdirClient.DeleteScope(s.Tx.Ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("deleting scope: %w", err)
	}
	return scopeRegistration, nil
}

func (s *scope) activate(scope scopes.Scope) (*types.ScopeRegistration, error) {
	scopeActivationPayload := clients.ToScopeRegistration(s.Tx.Instance, scope.CurrentScope)
	scopeRegistration, err := s.Tx.DigdirClient.ActivateScope(s.Tx.Ctx, scopeActivationPayload, scope.ToString())
	if err != nil {
		return nil, fmt.Errorf("acrivating scope: %w", err)
	}
	return scopeRegistration, nil
}

func (s *scope) Finalize(exposedScopes map[string]naisiov1.ExposedScope) error {
	filteredScopes, err := s.Tx.DigdirClient.GetFilteredScopes(s.Tx.Instance, s.Tx.Ctx, exposedScopes)
	if err != nil {
		return err
	}

	if filteredScopes != nil && len(filteredScopes.Current) > 0 {
		for _, scope := range filteredScopes.Current {
			s.Tx.Logger.Info(fmt.Sprintf("delete annotation set, deleting scope: %s from Maskinporten... ", scope.ToString()))
			if _, err := s.Tx.DigdirClient.DeleteScope(s.Tx.Ctx, scope.ToString()); err != nil {
				return fmt.Errorf("inactivating scope in Maskinporten: %w", err)
			}
			s.Rec.reportEvent(s.Tx, corev1.EventTypeNormal, EventDeactivatedScopeInDigDir, "Scope inactivated in Digdir")
		}
	}
	return nil
}
