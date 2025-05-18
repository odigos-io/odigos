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

// LatencySamplerSpec defines the desired state of a LatencySampler.
// This sampler filters traces based on HTTP route and latency thresholds.
// Traces with spans whose latency exceeds the specified threshold will be considered for sampling.
type LatencySamplerSpec struct {
	// ActionName is a user-defined identifier for this sampling action.
	// It can be used to reference this policy in UIs or configuration tools.
	ActionName string `json:"actionName,omitempty"`

	// Notes is an optional field for storing human-readable documentation or context for this sampler.
	Notes string `json:"notes,omitempty"`

	// Disabled indicates whether the sampler is currently active.
	// When true, this sampler will not be applied.
	Disabled bool `json:"disabled,omitempty"`

	// Signals lists the observability signal types (e.g., traces, metrics, logs)
	// that this sampler applies to.
	Signals []common.ObservabilitySignal `json:"signals"`

	// EndpointsFilters defines the list of route-based latency sampling filters.
	// Each filter targets a specific service and HTTP route with a latency threshold.
	// +kubebuilder:validation:Required
	EndpointsFilters []HttpRouteFilter `json:"endpoints_filters"`
}

// HttpRouteFilter defines a single latency-based sampling rule for an HTTP route.
type HttpRouteFilter struct {
	// HttpRoute is the route name (from span attribute "http.route") that this rule applies to.
	// +kubebuilder:validation:Required
	HttpRoute string `json:"http_route"`

	// ServiceName specifies the service that must emit the span for this rule to apply.
	// Matches the value of the "service.name" attribute in the span.
	// +kubebuilder:validation:Required
	ServiceName string `json:"service_name"`

	// MinimumLatencyThreshold is the latency in milliseconds that spans must exceed
	// to be considered for sampling. Spans with latency >= this value are eligible.
	// +kubebuilder:validation:Required
	MinimumLatencyThreshold int `json:"minimum_latency_threshold"`

	// FallbackSamplingRatio is the percentage (0–100) of traces to sample if the route
	// and service match but the span latency is below the threshold.
	// +kubebuilder:validation:Required
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

// LatencySamplerStatus defines the observed state of a LatencySampler.
// It captures runtime status such as readiness or deployment progress.
type LatencySamplerStatus struct {
	// Conditions contains the current status conditions for this sampler.
	// Typical types include "Available" and "Progressing".
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=latencysamplers,scope=Namespaced,shortName=ls
// +kubebuilder:metadata:labels=odigos.io/system-object=true
// +kubebuilder:storageversion
// LatencySampler is the Schema for defining latency-based trace sampling rules.
// It supports targeting specific services and HTTP routes and applying latency thresholds
// to determine sampling eligibility.
type LatencySampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LatencySamplerSpec   `json:"spec,omitempty"`
	Status LatencySamplerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// LatencySamplerList contains a list of LatencySampler objects.
type LatencySamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LatencySampler `json:"items"`
}
