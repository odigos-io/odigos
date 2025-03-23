package v1alpha1

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceNameSamplerSpec defines the desired state of ServiceNameSampler
type ServiceNameSamplerSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	// List of services to sample based on presence in the trace
	// +kubebuilder:validation:Required
	Services []ServiceNameFilter `json:"services"`
}

type ServiceNameFilter struct {
	// +kubebuilder:validation:Required
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

type ServiceNameSamplerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=servicenamesamplers,scope=Namespaced,shortName=sns
//+kubebuilder:metadata:labels=odigos.io/config=1
//+kubebuilder:metadata:labels=odigos.io/system-object=true

type ServiceNameSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceNameSamplerSpec   `json:"spec,omitempty"`
	Status ServiceNameSamplerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type ServiceNameSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceNameSampler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceNameSampler{}, &ServiceNameSamplerList{})
}
