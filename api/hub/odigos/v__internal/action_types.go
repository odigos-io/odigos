package v__internal

import (
	internalactions "github.com/odigos-io/odigos/api/hub/actions/v__internal"
	"github.com/odigos-io/odigos/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ActionSpec struct {
	ActionName string `json:"actionName,omitempty"`
	Notes      string `json:"notes,omitempty"`
	Disabled   bool   `json:"disabled,omitempty"`

	Signals []common.ObservabilitySignal `json:"signals"`

	AddClusterInfo  *internalactions.AddClusterInfoConfig  `json:"addClusterInfo,omitempty"`
	DeleteAttribute *internalactions.DeleteAttributeConfig `json:"deleteAttribute,omitempty"`
	RenameAttribute *internalactions.RenameAttributeConfig `json:"renameAttribute,omitempty"`
	PiiMasking      *internalactions.PiiMaskingConfig      `json:"piiMasking,omitempty"`
}

type ActionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type Action struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionSpec   `json:"spec,omitempty"`
	Status ActionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Action `json:"items"`
}

func (*Action) Hub() {}
