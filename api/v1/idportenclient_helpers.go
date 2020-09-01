package v1

import (
	"encoding/json"
	"fmt"
	"github.com/nais/digdirator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hash "github.com/mitchellh/hashstructure"
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
		return fmt.Errorf("failed to calculate application hash: %w", err)
	}
	in.Status.ProvisionHash = newHash
	return nil
}

func (in *IDPortenClient) HashUnchanged() (bool, error) {
	newHash, err := in.Hash()
	if err != nil {
		return false, fmt.Errorf("failed to calculate application hash: %w", err)
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
