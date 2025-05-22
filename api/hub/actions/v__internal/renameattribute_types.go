package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RenameAttributeConfig struct {
	Renames map[string]string `json:"renames"`
}

type RenameAttributeSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`
	Renames    map[string]string            `json:"renames"`
}

type RenameAttributeStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type RenameAttribute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RenameAttributeSpec   `json:"spec,omitempty"`
	Status RenameAttributeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type RenameAttributeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RenameAttribute `json:"items"`
}

func (*RenameAttribute) Hub() {}
