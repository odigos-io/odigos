/*
Copyright 2024.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataStream configures a group (or sub-pipeline) to export telemetry data from explicit sources to explicit destinations.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.streamName`
// +kubebuilder:printcolumn:name="Default",type=string,JSONPath=`.spec.default`
type DataStream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataStreamSpec   `json:"spec"`
	Status DataStreamStatus `json:"status,omitempty"`
}

type DataStreamSpec struct {
	// StreamName represents the name of the data stream.
	// This field is required upon creation and can be modified.
	// +kubebuilder:validation:Required
	StreamName string `json:"streamName"`
	// Default indicates whether this data stream is the default stream for objects that did not specifiy a stream name.
	// This field is optional and can be modified.
	// +kubebuilder:validation:Optional
	// +optional
	Default bool `json:"default,omitempty"`
}

type DataStreamStatus struct {
	// Represents the observations of a DataStream's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true

type DataStreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataStream `json:"items"`
}

// +kubebuilder:object:generate=false

func init() {
	SchemeBuilder.Register(&DataStream{}, &DataStreamList{})
}
