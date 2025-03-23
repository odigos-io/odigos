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

// SpanAttributeSamplerSpec defines the desired state of SpanAttributeSampler
type SpanAttributeSamplerSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	// Filters based on span attributes (e.g. env=prod, http.method, etc.)
	// +kubebuilder:validation:Required
	AttributesFilters []SpanAttributeFilter `json:"attributes_filters"`
}

type SpanAttributeFilter struct {
	// +kubebuilder:validation:Required
	AttributeKey string `json:"attribute_key"`

	// Supported: exists, equals, not_equals
	// +kubebuilder:validation:Enum=exists;equals;not_equals
	// +kubebuilder:validation:Required
	Condition string `json:"condition"`

	// Optional: only required for equals and not_equals
	ExpectedValue string `json:"expected_value,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

type SpanAttributeSamplerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=spanattributesamplers,scope=Namespaced,shortName=sas
//+kubebuilder:metadata:labels=odigos.io/config=1
//+kubebuilder:metadata:labels=odigos.io/system-object=true

type SpanAttributeSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpanAttributeSamplerSpec   `json:"spec,omitempty"`
	Status SpanAttributeSamplerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type SpanAttributeSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpanAttributeSampler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpanAttributeSampler{}, &SpanAttributeSamplerList{})
}
