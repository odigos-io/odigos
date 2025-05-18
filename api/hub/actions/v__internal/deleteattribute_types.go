package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeleteAttributeConfig struct {
	AttributeNamesToDelete []string `json:"attributeNamesToDelete"`
}

type DeleteAttributeSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	AttributeNamesToDelete []string `json:"attributeNamesToDelete"`
}

type DeleteAttributeStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type DeleteAttribute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeleteAttributeSpec   `json:"spec,omitempty"`
	Status DeleteAttributeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DeleteAttributeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeleteAttribute `json:"items"`
}

func (*DeleteAttribute) Hub() {}
