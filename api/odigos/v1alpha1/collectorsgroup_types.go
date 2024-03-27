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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=CLUSTER_GATEWAY;NODE_COLLECTOR
type CollectorsGroupRole string

const (
	CollectorsGroupRoleClusterGateway CollectorsGroupRole = "CLUSTER_GATEWAY"
	CollectorsGroupRoleNodeCollector  CollectorsGroupRole = "NODE_COLLECTOR"
)

// CollectorsGroupSpec defines the desired state of Collector
type CollectorsGroupSpec struct {
	InputSvc string              `json:"inputSvc,omitempty"`
	Role     CollectorsGroupRole `json:"role"`
}

// CollectorsGroupStatus defines the observed state of Collector
type CollectorsGroupStatus struct {
	Ready bool `json:"ready,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CollectorsGroup is the Schema for the collectors API
type CollectorsGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CollectorsGroupSpec   `json:"spec,omitempty"`
	Status CollectorsGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CollectorsGroupList contains a list of Collector
type CollectorsGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CollectorsGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CollectorsGroup{}, &CollectorsGroupList{})
}
