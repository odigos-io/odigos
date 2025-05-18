package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LatencySamplerSpec struct {
	ActionName       string                       `json:"actionName,omitempty"`
	Notes            string                       `json:"notes,omitempty"`
	Disabled         bool                         `json:"disabled,omitempty"`
	Signals          []common.ObservabilitySignal `json:"signals"`
	EndpointsFilters []HttpRouteFilter            `json:"endpoints_filters"`
}

type HttpRouteFilter struct {
	HttpRoute               string  `json:"http_route"`
	ServiceName             string  `json:"service_name"`
	MinimumLatencyThreshold int     `json:"minimum_latency_threshold"`
	FallbackSamplingRatio   float64 `json:"fallback_sampling_ratio"`
}

type LatencySamplerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type LatencySampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              LatencySamplerSpec   `json:"spec,omitempty"`
	Status            LatencySamplerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type LatencySamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LatencySampler `json:"items"`
}

func (*LatencySampler) Hub() {}
