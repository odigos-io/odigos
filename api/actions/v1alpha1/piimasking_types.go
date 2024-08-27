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

// +kubebuilder:validation:Enum=CREDIT_CARD
type PiiCategory string

const (
	CreditCardMasking PiiCategory = "CREDIT_CARD"
)

// PiiMaskingSpec defines the desired state of PiiMasking action
type PiiMaskingSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	PiiCategories []PiiCategory `json:"piiCategories"`
}

// PiiMaskingStatus defines the observed state of PiiMasking action
type PiiMaskingStatus struct {
	// Represents the observations of a piiMasking's current state.
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
//+kubebuilder:resource:path=piimaskings,scope=Namespaced,shortName=red
//+kubebuilder:metadata:labels=odigos.io/config=1
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// PiiMasking is the Schema for the PiiMasking odigos action API
type PiiMasking struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PiiMaskingSpec   `json:"spec,omitempty"`
	Status PiiMaskingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PiiMaskingList contains a list of PiiMasking
type PiiMaskingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PiiMasking `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PiiMasking{}, &PiiMaskingList{})
}
