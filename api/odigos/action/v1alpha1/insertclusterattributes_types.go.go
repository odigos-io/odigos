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
	"github.com/keyval-dev/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OtelAttributeWithValue struct {

	// the name of the attribute to insert
	AttributeName string `json:"attributeName"`

	// if the value is a string, this field should be used.
	// empty string is a valid value
	AttributeStringValue *string `json:"attributeValue,omitempty"`
}

// InsertClusterAttributesSpec defines the desired state of InsertClusterAttributes action
type InsertClusterAttributesSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	ClusterAttributes []OtelAttributeWithValue `json:"clusterAttributes"`
}

// InsertClusterAttributesStatus defines the observed state of InsertClusterAttributes action
type InsertClusterAttributesStatus struct {
	// Represents the observations of a insertclusterattributes's current state.
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

// InsertClusterAttributes is the Schema for the insertclusterattributes odigos action API
type InsertClusterAttributes struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InsertClusterAttributesSpec   `json:"spec,omitempty"`
	Status InsertClusterAttributesStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InsertClusterAttributesList contains a list of InsertClusterAttributes
type InsertClusterAttributesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InsertClusterAttributes `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InsertClusterAttributes{}, &InsertClusterAttributesList{})
}
