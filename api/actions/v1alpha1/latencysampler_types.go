/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LatencySamplerSpec defines the desired state of LatencySampler action
type LatencySamplerSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	// Specifies the list of endpoint filters to be applied for sampling
	// +kubebuilder:validation:Required
	EndpointsFilters []HttpRouteFilter `json:"endpoints_filters"`
}

type HttpRouteFilter struct {
	// Specifies the http.route to be sampled
	// +kubebuilder:validation:Required
	HttpRoute string `json:"http_route"`
	// Specifies the service to be sampled
	// +kubebuilder:validation:Required
	ServiceName string `json:"service_name"`
	// Specifies the lower latency threshold in milliseconds; traces with latency equal to or exceeding this value will be considered for sampling.
	// +kubebuilder:validation:Required
	MinimumLatencyThreshold int `json:"minimum_latency_threshold"`
	// Specifies the fallback sampling ratio to be applied in case service and endpoint filter match but the latency threshold is not met.
	// +kubebuilder:validation:Required
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

// LatencySamplerStatus defines the observed state of LatencySampler action
type LatencySamplerStatus struct {
	// Represents the observations of a LatencySampler's current state.
	// Known .status.conditions.type are: "Available", "Progressing"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=latencysamplers,scope=Namespaced,shortName=ls
//+kubebuilder:metadata:labels=odigos.io/config=1
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// LatencySampler is the Schema for the LatencySampler odigos action API
type LatencySampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LatencySamplerSpec   `json:"spec,omitempty"`
	Status LatencySamplerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LatencySamplerList contains a list of LatencySampler
type LatencySamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LatencySampler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LatencySampler{}, &LatencySamplerList{})
}
