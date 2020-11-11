package v1

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/hashstructure"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:generate=false
type Instance interface {
	metav1.Object
	runtime.Object
	CalculateHash() (string, error)
	IsHashUnchanged() (bool, error)
	GetIntegrationType() types.IntegrationType
	GetSecretName() string
	GetStatus() *ClientStatus
	MakeLabels() map[string]string
	MakeDescription() string
	ToClientRegistration() types.ClientRegistration
}

// ClientStatus defines the observed state of Current Client
type ClientStatus struct {
	// ClientID is the current Client ID
	ClientID string `json:"clientID"`
	// Timestamp is the last time the Status subresource was updated
	Timestamp metav1.Time `json:"timestamp,omitempty"`
	// ProvisionHash is the hash of the Current Client object
	ProvisionHash string `json:"provisionHash,omitempty"`
	// CorrelationID is the ID referencing the processing transaction last performed on this resource
	CorrelationID string `json:"correlationID"`
	// KeyIDs is the list of key IDs for valid JWKs registered for the client at Digdir
	KeyIDs []string `json:"keyIDs"`
}

func (in *ClientStatus) GetHash() string {
	return in.ProvisionHash
}

func (in *ClientStatus) SetHash(hash string) {
	in.Timestamp = metav1.Now()
	in.ProvisionHash = hash
}

func (in *ClientStatus) GetClientID() string {
	return in.ClientID
}

func InstanceIsBeingDeleted(instance Instance) bool {
	return !instance.GetDeletionTimestamp().IsZero()
}

func HasFinalizer(instance Instance, finalizerName string) bool {
	return util.ContainsString(instance.GetFinalizers(), finalizerName)
}

// MakeDescription returns a description that identifies an application in NAIS
func MakeDescription(instance Instance) string {
	return fmt.Sprintf("%s:%s:%s", instance.GetClusterName(), instance.GetNamespace(), instance.GetName())
}

func CalculateHash(in interface{}) (string, error) {
	marshalled, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf("marshalling input: %w", err)
	}
	h, err := hashstructure.Hash(marshalled, nil)
	return fmt.Sprintf("%x", h), err
}

func IsHashUnchanged(instance Instance) (bool, error) {
	previousHash := instance.GetStatus().GetHash()
	currentHash, err := instance.CalculateHash()
	if err != nil {
		return false, fmt.Errorf("calculating hash while comparing hashes: %w", err)
	}
	return previousHash == currentHash, nil
}