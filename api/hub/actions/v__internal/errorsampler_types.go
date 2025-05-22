package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ErrorSamplerSpec struct {
	ActionName            string                       `json:"actionName,omitempty"`
	Notes                 string                       `json:"notes,omitempty"`
	Disabled              bool                         `json:"disabled,omitempty"`
	Signals               []common.ObservabilitySignal `json:"signals"`
	FallbackSamplingRatio float64                      `json:"fallback_sampling_ratio"`
}

type ErrorSamplerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type ErrorSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ErrorSamplerSpec   `json:"spec,omitempty"`
	Status            ErrorSamplerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ErrorSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ErrorSampler `json:"items"`
}

func (*ErrorSampler) Hub() {}
