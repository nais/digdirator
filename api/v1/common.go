package v1

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/hashstructure"
	"github.com/nais/digdirator/pkg/digdir/types"
	"github.com/nais/digdirator/pkg/util"
	"gopkg.in/square/go-jose.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +kubebuilder:object:generate=false
type Instance interface {
	metav1.Object
	runtime.Object
	schema.ObjectKind
	CalculateHash() (string, error)
	CreateSecretData(jose.JSONWebKey) (map[string]string, error)
	HasFinalizer(string) bool
	IsBeingDeleted() bool
	IsUpToDate() (bool, error)
	GetIntegrationType() types.IntegrationType
	GetInstanceType() string
	GetSecretMapKey() string
	GetSecretName() string
	GetStatus() *ClientStatus
	MakeLabels() map[string]string
	MakeDescription() string
	ToClientRegistration() types.ClientRegistration
}

// ClientStatus defines the observed state of Current Client
type ClientStatus struct {
	// SynchronizationState denotes the last known state of the Instance during synchronization
	SynchronizationState string `json:"synchronizationState,omitempty"`
	// SynchronizationTime is the last time the Status subresource was updated
	SynchronizationTime *metav1.Time `json:"synchronizationTime,omitempty"`
	// SynchronizationHash is the hash of the Instance object
	SynchronizationHash string `json:"synchronizationHash,omitempty"`
	// ClientID is the corresponding client ID for this client at Digdir
	ClientID string `json:"clientID,omitempty"`
	// CorrelationID is the ID referencing the processing transaction last performed on this resource
	CorrelationID string `json:"correlationID,omitempty"`
	// KeyIDs is the list of key IDs for valid JWKs registered for the client at Digdir
	KeyIDs []string `json:"keyIDs,omitempty"`
}

func (in *ClientStatus) GetSynchronizationHash() string {
	return in.SynchronizationHash
}

func (in *ClientStatus) SetHash(hash string) {
	in.SynchronizationHash = hash
}

func (in *ClientStatus) SetStateSynchronized() {
	now := metav1.Now()
	in.SynchronizationTime = &now
	in.SynchronizationState = EventSynchronized
}

func (in *ClientStatus) GetClientID() string {
	return in.ClientID
}

func (in *ClientStatus) SetClientID(clientID string) {
	in.ClientID = clientID
}

func (in *ClientStatus) SetCorrelationID(correlationID string) {
	in.CorrelationID = correlationID
}

func (in *ClientStatus) GetKeyIDs() []string {
	return in.KeyIDs
}

func (in *ClientStatus) SetKeyIDs(keyIDs []string) {
	in.KeyIDs = keyIDs
}

func (in *ClientStatus) SetSynchronizationState(state string) {
	in.SynchronizationState = state
}

func isBeingDeleted(instance Instance) bool {
	return !instance.GetDeletionTimestamp().IsZero()
}

func hasFinalizer(instance Instance, finalizerName string) bool {
	return util.ContainsString(instance.GetFinalizers(), finalizerName)
}

// makeDescription returns a description that identifies an application in NAIS
func makeDescription(instance Instance) string {
	return fmt.Sprintf("%s:%s:%s", instance.GetClusterName(), instance.GetNamespace(), instance.GetName())
}

func calculateHash(in interface{}) (string, error) {
	marshalled, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf("marshalling input: %w", err)
	}
	h, err := hashstructure.Hash(marshalled, nil)
	return fmt.Sprintf("%x", h), err
}

func isHashUnchanged(instance Instance) (bool, error) {
	previousHash := instance.GetStatus().GetSynchronizationHash()
	currentHash, err := instance.CalculateHash()
	if err != nil {
		return false, fmt.Errorf("calculating hash while comparing hashes: %w", err)
	}
	return previousHash == currentHash, nil
}

func isUpToDate(instance Instance) (bool, error) {
	hashUnchanged, err := isHashUnchanged(instance)
	if err != nil {
		return false, err
	}
	return hashUnchanged, nil
}
