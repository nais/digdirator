package v1

// +groupName="nais.io"

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=maskinportenclient

// MaskinportenClient is the Schema for the MaskinportenClient API
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.spec.secretName`
// +kubebuilder:printcolumn:name="ClientID",type=string,JSONPath=`.status.clientID`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type MaskinportenClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MaskinportenClientSpec `json:"spec,omitempty"`
	Status ClientStatus           `json:"status,omitempty"`
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
	Scopes []MaskinportenScope `json:"scopes"`
	// SecretName is the name of the resulting Secret resource to be created
	SecretName string `json:"secretName"`
}

type MaskinportenScope struct {
	Scope string `json:"scope"`
}

func init() {
	SchemeBuilder.Register(&MaskinportenClient{}, &MaskinportenClientList{})
}
