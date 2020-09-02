package v1

// +groupName="nais.io"

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=idportenclient
// +kubebuilder:subresource:status

// IDPortenClient is the Schema for the IDPortenClients API
// +kubebuilder:printcolumn:name="Secret",type=string,JSONPath=`.spec.secretName`
// +kubebuilder:printcolumn:name="ClientId",type=string,JSONPath=`.status.clientId`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type IDPortenClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IDPortenClientSpec   `json:"spec,omitempty"`
	Status IDPortenClientStatus `json:"status,omitempty"`
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
	// ClientName is the name of the client registered at DigDir
	ClientName string `json:"clientName,omitempty"`
	// ClientURI is the URL to the client to be used at DigDir when displaying a 'back' button or on errors
	ClientURI string `json:"clientURI,omitempty"`
	// ReplyURLs is a list of reply URLs to be registered at DigDir
	ReplyURLs []string `json:"replyURLs"`
	// SecretName is the name of the resulting Secret resource to be created
	SecretName string `json:"secretName"`
	// FrontchannelLogoutURI is the URL that ID-porten sends a requests to whenever a logout is triggered by another application using the same session
	FrontchannelLogoutURI string `json:"frontchannelLogoutURI"`
	// PostLogoutRedirectURI is a list of valid URIs that ID-porten may redirect to after logout
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs"`
	// RefreshTokenLifetime is the lifetime in seconds for the issued refresh token from ID-porten
	RefreshTokenLifetime int `json:"refreshTokenLifetime"`
	// Scopes is a list of valid scopes that the client may request
	Scopes []string `json:"scopes"`
	// TokenEndpointAuthMethod is the preferred authentication method for the client.
	// +kubebuilder:validation:Enum=client_secret_post;client_secret_basic;private_key_jwt;none
	TokenEndpointAuthMethod string `json:"tokenEndpointAuthMethod"`
}

// IDPortenClientStatus defines the observed state of IDPortenClient
type IDPortenClientStatus struct {
	// ClientId is the ID-porten client ID
	ClientId string `json:"clientId"`
	// Timestamp is the last time the Status subresource was updated
	Timestamp metav1.Time `json:"timestamp,omitempty"`
	// ProvisionHash is the hash of the IDPortenClient object
	ProvisionHash string `json:"provisionHash,omitempty"`
	// CorrelationId is the ID referencing the processing transaction last performed on this resource
	CorrelationId string `json:"correlationId"`
}

func init() {
	SchemeBuilder.Register(&IDPortenClient{}, &IDPortenClientList{})
}
