package v1

// +groupName="nais.io"

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=maskinportenclient
// +kubebuilder:subresource:status

// MaskinportenClient is the Schema for the MaskinportenClient API
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.spec.secretName`
// +kubebuilder:printcolumn:name="ClientID",type=string,JSONPath=`.status.clientID`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type MaskinportenClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MaskinportenClientSpec   `json:"spec,omitempty"`
	Status            MaskinportenClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MaskinportenClientList contains a list of MaskinportenClient
type MaskinportenClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MaskinportenClient `json:"items"`
}

// MaskinportenClientSpec defines the desired state of MaskinportenClient
type MaskinportenClientSpec struct {
	// Scopes is a list of valid scopes that the client can request tokens for
	Scopes []string `json:"scopes"`
}

// MaskinportenClientStatus defines the observed state of MaskinportenClient
type MaskinportenClientStatus struct {
	// ClientID is the Maskinporten client ID
	ClientID string `json:"clientID"`
	// Timestamp is the last time the Status subresource was updated
	Timestamp metav1.Time `json:"timestamp,omitempty"`
	// ProvisionHash is the hash of the MaskinportenClient object
	ProvisionHash string `json:"provisionHash,omitempty"`
	// CorrelationID is the ID referencing the processing transaction last performed on this resource
	CorrelationID string `json:"correlationID"`
	// KeyIDs is the list of key IDs for valid JWKs registered for the client at Digdir
	KeyIDs []string `json:"keyIDs"`
}

func init() {
	SchemeBuilder.Register(&MaskinportenClient{}, &MaskinportenClientList{})
}
