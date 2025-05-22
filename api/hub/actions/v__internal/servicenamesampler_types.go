package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceNameSamplerSpec struct {
	ActionName          string                       `json:"actionName,omitempty"`
	Notes               string                       `json:"notes,omitempty"`
	Disabled            bool                         `json:"disabled,omitempty"`
	Signals             []common.ObservabilitySignal `json:"signals"`
	ServicesNameFilters []ServiceNameFilter          `json:"services_name_filters"`
}

type ServiceNameFilter struct {
	ServiceName           string  `json:"service_name"`
	SamplingRatio         float64 `json:"sampling_ratio"`
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

type ServiceNameSamplerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type ServiceNameSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceNameSamplerSpec   `json:"spec,omitempty"`
	Status ServiceNameSamplerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ServiceNameSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceNameSampler `json:"items"`
}

func (*ServiceNameSampler) Hub() {}
