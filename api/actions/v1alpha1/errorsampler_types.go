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

// ErrorSamplerSpec defines the desired state of ErrorSampler action
type ErrorSamplerSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	// Specifies the ratio of non-error traces to be sampled.
	// +kubebuilder:validation:Required
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

// ErrorSamplerStatus defines the observed state of ErrorSampler action
type ErrorSamplerStatus struct {
	// Represents the observations of a ErrorSampler's current state.
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
//+kubebuilder:resource:path=errorsamplers,scope=Namespaced,shortName=es
//+kubebuilder:metadata:labels=odigos.io/config=1
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// ErrorSampler is the Schema for the ErrorSampler odigos action API
type ErrorSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ErrorSamplerSpec   `json:"spec,omitempty"`
	Status ErrorSamplerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ErrorSamplerList contains a list of ErrorSampler
type ErrorSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ErrorSampler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ErrorSampler{}, &ErrorSamplerList{})
}
