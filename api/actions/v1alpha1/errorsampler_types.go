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

// ErrorSamplerSpec defines the configuration for an ErrorSampler action.
// This sampler prioritizes traces that contain errors, and allows specifying a fallback sampling ratio
// for traces that do not include any errors.
type ErrorSamplerSpec struct {
	// ActionName is an optional identifier for this sampler rule.
	// It can be used for referencing, labeling, or displaying the rule in UIs.
	ActionName string `json:"actionName,omitempty"`

	// Notes provides free-form documentation or context for the user.
	Notes string `json:"notes,omitempty"`

	// Disabled indicates whether the sampler is currently active.
	// When true, the sampler will not be evaluated or applied.
	Disabled bool `json:"disabled,omitempty"`

	// Signals specifies the types of telemetry data this sampler should apply to.
	// Typically, this includes "traces", but may also include "logs" or "metrics".
	Signals []common.ObservabilitySignal `json:"signals"`

	// FallbackSamplingRatio determines the percentage (0–100) of non-error traces
	// that should be sampled. Error traces are always sampled.
	// +kubebuilder:validation:Required
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

// ErrorSamplerStatus defines the runtime state and observed conditions of an ErrorSampler.
// It may include conditions such as "Available" or "Progressing".
type ErrorSamplerStatus struct {
	// Conditions captures the current operational state of the sampler.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=errorsamplers,scope=Namespaced,shortName=es
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// ErrorSampler is the Schema for the ErrorSampler CRD.
// It defines sampling logic that always retains traces with errors, and optionally samples
// non-error traces based on the fallback ratio.
type ErrorSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ErrorSamplerSpec   `json:"spec,omitempty"`
	Status ErrorSamplerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ErrorSamplerList contains a list of ErrorSampler resources.
type ErrorSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ErrorSampler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ErrorSampler{}, &ErrorSamplerList{})
}
