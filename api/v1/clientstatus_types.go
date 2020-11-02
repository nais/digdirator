package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
