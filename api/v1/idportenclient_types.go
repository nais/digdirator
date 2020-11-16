package v1

// +groupName="nais.io"

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=idportenclient

// IDPortenClient is the Schema for the IDPortenClients API
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.spec.secretName`
// +kubebuilder:printcolumn:name="ClientID",type=string,JSONPath=`.status.clientID`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type IDPortenClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IDPortenClientSpec `json:"spec,omitempty"`
	Status ClientStatus       `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IDPortenClientList contains a list of IDPortenClient
type IDPortenClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IDPortenClient `json:"items"`
}

// IDPortenClientSpec defines the desired state of IDPortenClient
type IDPortenClientSpec struct {
	// ClientURI is the URL to the client to be used at DigDir when displaying a 'back' button or on errors
	ClientURI string `json:"clientURI"`
	// RedirectURI is the redirect URI to be registered at DigDir
	// +kubebuilder:validation:Pattern=`^https:\/\/`
	RedirectURI string `json:"redirectURI"`
	// SecretName is the name of the resulting Secret resource to be created
	SecretName string `json:"secretName"`
	// FrontchannelLogoutURI is the URL that ID-porten sends a requests to whenever a logout is triggered by another application using the same session
	FrontchannelLogoutURI string `json:"frontchannelLogoutURI,omitempty"`
	// PostLogoutRedirectURI is a list of valid URIs that ID-porten may redirect to after logout
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	// SessionLifetime is the maximum session lifetime in seconds for a logged in end-user for this client.
	// +kubebuilder:validation:Minimum=3600
	// +kubebuilder:validation:Maximum=43200
	SessionLifetime int `json:"sessionLifetime"`
}

func init() {
	SchemeBuilder.Register(&IDPortenClient{}, &IDPortenClientList{})
}
