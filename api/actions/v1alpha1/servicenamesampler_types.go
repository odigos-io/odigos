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

// ServiceNameSamplerSpec defines the desired configuration for a ServiceNameSampler.
// This sampler enables sampling based on the presence of specific service names within a trace.
type ServiceNameSamplerSpec struct {
	// ActionName is an optional label to identify this sampling policy.
	// It can be used for display purposes or integration with other tools.
	ActionName string `json:"actionName,omitempty"`

	// Notes allows attaching additional free-form documentation or context to this sampler.
	Notes string `json:"notes,omitempty"`

	// Disabled indicates whether this sampler should be active.
	// If true, the sampler will not be applied.
	Disabled bool `json:"disabled,omitempty"`

	// Signals specifies which types of telemetry data this sampler applies to.
	// Common values include "traces", "metrics", or "logs".
	Signals []common.ObservabilitySignal `json:"signals"`

	// ServicesNameFilters defines rules for sampling traces based on the presence
	// of specific service names. If a trace contains a span from one of the listed
	// services, the associated sampling ratio is applied.
	// +kubebuilder:validation:Required
	ServicesNameFilters []ServiceNameFilter `json:"services_name_filters"`
}

// ServiceNameFilter defines a single rule that maps a service name to a sampling decision.
type ServiceNameFilter struct {
	// ServiceName specifies the name of the service to look for within a trace.
	// If any span in the trace comes from this service, the rule will apply.
	// +kubebuilder:validation:Required
	ServiceName string `json:"service_name"`

	// SamplingRatio determines the percentage (0–100) of traces to sample
	// when the specified service is present in the trace.
	//
	// For example, a value of 100 means all such traces will be kept,
	// while a value of 0 means all will be dropped.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=100
	SamplingRatio float64 `json:"sampling_ratio"`

	// FallbackSamplingRatio is the percentage (0–100) of traces to sample
	// if the specified service is not present in the trace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=100
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

// ServiceNameSamplerStatus represents the runtime status of a ServiceNameSampler,
// including observed conditions such as validation errors or processing state.
type ServiceNameSamplerStatus struct {
	// Conditions is a list of status conditions for this sampler,
	// following the standard Kubernetes conventions.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=servicenamesamplers,scope=Namespaced,shortName=sns
// +kubebuilder:metadata:labels=odigos.io/system-object=true

// ServiceNameSampler is the Schema for the servicenamesamplers API.
// It enables trace sampling based on whether specific services appear within a trace.
type ServiceNameSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceNameSamplerSpec   `json:"spec,omitempty"`
	Status ServiceNameSamplerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceNameSamplerList contains a list of ServiceNameSampler resources.
type ServiceNameSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceNameSampler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceNameSampler{}, &ServiceNameSamplerList{})
}
