package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PiiCategory string

type PiiMaskingConfig struct {
	PiiCategories []PiiCategory `json:"piiCategories"`
}

type PiiMaskingSpec struct {
	ActionName    string                       `json:"actionName,omitempty"`
	Notes         string                       `json:"notes,omitempty"`
	Disabled      bool                         `json:"disabled,omitempty"`
	Signals       []common.ObservabilitySignal `json:"signals"`
	PiiCategories []PiiCategory                `json:"piiCategories"`
}

type PiiMaskingStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type PiiMasking struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PiiMaskingSpec   `json:"spec,omitempty"`
	Status PiiMaskingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type PiiMaskingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PiiMasking `json:"items"`
}

func (*PiiMasking) Hub() {}
