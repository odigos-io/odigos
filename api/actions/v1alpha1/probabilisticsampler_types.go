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

// ProbablisticSamplerSpec defines the desired state of ProbablisticSampler
type ProbabilisticSamplerSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	SamplingPercentage string `json:"sampling_percentage"`
}

// ProbablisticSamplerStatus defines the observed state of ProbablisticSampler
type ProbabilisticSamplerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=probabilisticsampler,scope=Namespaced,shortName=ps
// ProbablisticSampler is the Schema for the ProbablisticSampler odigos action API
type ProbabilisticSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProbabilisticSamplerSpec   `json:"spec,omitempty"`
	Status ProbabilisticSamplerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProbabilisticSamplerList contains a list of ProbabilisticSampler

type ProbabilisticSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProbabilisticSampler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProbabilisticSampler{}, &ProbabilisticSamplerList{})
}
