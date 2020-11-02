package v1

import (
	"encoding/json"
	"fmt"
	hash "github.com/mitchellh/hashstructure"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const Maskinporten = "maskinporten"

func (in *MaskinportenClient) IsBeingDeleted() bool {
	return !in.ObjectMeta.DeletionTimestamp.IsZero()
}

func (in *MaskinportenClient) HasFinalizer(finalizerName string) bool {
	return util.ContainsString(in.ObjectMeta.Finalizers, finalizerName)
}

func (in *MaskinportenClient) AddFinalizer(finalizerName string) {
	in.ObjectMeta.Finalizers = append(in.ObjectMeta.Finalizers, finalizerName)
}

func (in *MaskinportenClient) RemoveFinalizer(finalizerName string) {
	in.ObjectMeta.Finalizers = util.RemoveString(in.ObjectMeta.Finalizers, finalizerName)
}

func (in *MaskinportenClient) UpdateHash() error {
	in.Status.Timestamp = metav1.Now()
	newHash, err := in.Hash()
	if err != nil {
		return fmt.Errorf("calculating application hash: %w", err)
	}
	in.Status.ProvisionHash = newHash
	return nil
}

func (in *MaskinportenClient) ClientDescription() string {
	return fmt.Sprintf("%s:%s:%s", in.ClusterName, in.Namespace, in.Name)
}

func (in *MaskinportenClient) HashUnchanged() (bool, error) {
	newHash, err := in.Hash()
	if err != nil {
		return false, fmt.Errorf("checking if hash is unchanged: %w", err)
	}
	return in.Status.ProvisionHash == newHash, nil
}

func (in MaskinportenClient) Hash() (string, error) {
	marshalled, err := json.Marshal(in.Spec)
	if err != nil {
		return "", err
	}
	h, err := hash.Hash(marshalled, nil)
	return fmt.Sprintf("%x", h), err
}

func (in MaskinportenClient) GetUniqueName() string {
	return fmt.Sprintf("%s:%s:%s:%s", Maskinporten, in.ClusterName, in.Namespace, in.Name)
}

func (in MaskinportenClient) ToClientRegistration() types.ClientRegistration {
	return types.ClientRegistration{
		AccessTokenLifetime:               3600,
		ApplicationType:                   types.ApplicationTypeWeb,
		AuthorizationLifeTime:             0,
		ClientName:                        types.DefaultClientName,
		ClientURI:                         "",
		Description:                       in.GetUniqueName(),
		FrontchannelLogoutSessionRequired: false,
		FrontchannelLogoutURI:             "",
		GrantTypes: []types.GrantType{
			types.GrantTypeJwtBearer,
		},
		IntegrationType:         types.IntegrationTypeMaskinporten,
		PostLogoutRedirectURIs:  nil,
		RedirectURIs:            nil,
		RefreshTokenLifetime:    0,
		RefreshTokenUsage:       types.RefreshTokenUsageOneTime,
		Scopes:                  in.Spec.Scopes,
		TokenEndpointAuthMethod: types.TokenEndpointAuthMethodPrivateKeyJwt,
	}
}
