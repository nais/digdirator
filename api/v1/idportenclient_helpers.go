package v1

import (
	"encoding/json"
	"fmt"
	hash "github.com/mitchellh/hashstructure"
	"github.com/nais/digdirator/pkg/idporten"
	"github.com/nais/digdirator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (in *IDPortenClient) IsBeingDeleted() bool {
	return !in.ObjectMeta.DeletionTimestamp.IsZero()
}

func (in *IDPortenClient) HasFinalizer(finalizerName string) bool {
	return util.ContainsString(in.ObjectMeta.Finalizers, finalizerName)
}

func (in *IDPortenClient) AddFinalizer(finalizerName string) {
	in.ObjectMeta.Finalizers = append(in.ObjectMeta.Finalizers, finalizerName)
}

func (in *IDPortenClient) RemoveFinalizer(finalizerName string) {
	in.ObjectMeta.Finalizers = util.RemoveString(in.ObjectMeta.Finalizers, finalizerName)
}

func (in *IDPortenClient) UpdateHash() error {
	in.Status.Timestamp = metav1.Now()
	newHash, err := in.Hash()
	if err != nil {
		return fmt.Errorf("calculating application hash: %w", err)
	}
	in.Status.ProvisionHash = newHash
	return nil
}

func (in *IDPortenClient) ClientID() string {
	return fmt.Sprintf("%s:%s:%s", in.ClusterName, in.Namespace, in.Name)
}

func (in *IDPortenClient) HashUnchanged() (bool, error) {
	newHash, err := in.Hash()
	if err != nil {
		return false, fmt.Errorf("checking if hash is unchanged: %w", err)
	}
	return in.Status.ProvisionHash == newHash, nil
}

func (in IDPortenClient) Hash() (string, error) {
	marshalled, err := json.Marshal(in.Spec)
	if err != nil {
		return "", err
	}
	h, err := hash.Hash(marshalled, nil)
	return fmt.Sprintf("%x", h), err
}

func (in IDPortenClient) GetUniqueName() string {
	return fmt.Sprintf("%s:%s:%s", in.ClusterName, in.Namespace, in.Name)
}

func (in IDPortenClient) ToClientRegistration() idporten.ClientRegistration {
	return idporten.ClientRegistration{
		AccessTokenLifetime:               3600,
		ApplicationType:                   idporten.ApplicationTypeWeb,
		AuthorizationLifeTime:             0,
		ClientName:                        in.Spec.ClientName,
		ClientURI:                         in.Spec.ClientURI,
		Description:                       in.GetUniqueName(),
		FrontchannelLogoutSessionRequired: false,
		FrontchannelLogoutURI:             in.Spec.FrontchannelLogoutURI,
		GrantTypes: []idporten.GrantType{
			idporten.GrantTypeAuthorizationCode,
		},
		IntegrationType:         idporten.IntegrationTypeIDPorten,
		PostLogoutRedirectURIs:  in.Spec.PostLogoutRedirectURIs,
		RedirectURIs:            in.Spec.ReplyURLs,
		RefreshTokenLifetime:    12 * 3600,
		RefreshTokenUsage:       idporten.RefreshTokenUsageOneTime,
		Scopes:                  in.Spec.Scopes,
		TokenEndpointAuthMethod: idporten.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}
