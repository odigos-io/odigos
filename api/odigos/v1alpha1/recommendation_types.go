/*
Copyright 2026.

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

// RecommendationSpec defines the desired state of Recommendation
type RecommendationSpec struct {
	// Type identifies which recommendation from the catalog this resource enables.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=InferDBAttributes;AutoGoOffsetUpdater;EnableOwnMetrics;SampleHealthProbes;UrlTemplatization
	Type common.RecommendationType `json:"type"`

	// Applied indicates whether this recommendation is currently applied
	// (via the recommendation flow or configured organically).
	// +optional
	Applied bool `json:"applied,omitempty"`

	// ConditionsMet indicates whether the catalog prerequisites for this recommendation
	// currently hold (for example GoEnterpriseSources for AutoGoOffsetUpdater).
	// +optional
	ConditionsMet bool `json:"conditionsMet,omitempty"`
}

// RecommendationStatus defines the observed state of Recommendation
type RecommendationStatus struct {
	// Represents the observations of a recommendation's current state.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:metadata:labels=odigos.io/system-object=true
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`

// Recommendation is the Schema for enabling a catalog recommendation in the cluster.
type Recommendation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RecommendationSpec   `json:"spec,omitempty"`
	Status RecommendationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RecommendationList contains a list of Recommendation
type RecommendationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Recommendation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Recommendation{}, &RecommendationList{})
}
