package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CertMerge is the Schema for the certmerges API
// +k8s:openapi-gen=true
type CertMerge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CertMergeSpec   `json:"spec,omitempty"`
	Status CertMergeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CertMergeList contains a list of CertMerge
type CertMergeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CertMerge `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CertMerge{}, &CertMergeList{})
}

// CertMergeSpec defines the desired state of CertMerge
type CertMergeSpec struct {
	SecretName      string             `json:"name"`
	Selector        []SecretSelector   `json:"selector"`
	SecretNamespace string             `json:"namespace"`
	SecretList      []SecretDefinition `json:"secretlist"`
}

// SecretSelector defines the needed parameters to search for secrets by Label
type SecretSelector struct {
	LabelSelector metav1.LabelSelector `json:"labelselector"`
	Namespace     string               `json:"namespace"`
}

// SecretDefinition defines the parameters to search for secrets by name
type SecretDefinition struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// CertMergeStatus defines the observed state of CertMerge
type CertMergeStatus struct {
	UpToDate         bool        `json:"uptodate"`
	Version          string      `json:"version,omitempty"`
	Items            []string    `json:"items,omitempty"`
	UpdatedTimestamp metav1.Time `json:"updatedTimestamp,omitempty"`
}
