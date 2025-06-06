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
	"github.com/odigos-io/odigos/common/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DestinationSpec defines the desired state of Destination
type DestinationSpec struct {
	Type            common.DestinationType       `json:"type"`
	DestinationName string                       `json:"destinationName"`
	Data            map[string]string            `json:"data"`
	SecretRef       *v1.LocalObjectReference     `json:"secretRef,omitempty"`
	Signals         []common.ObservabilitySignal `json:"signals"`

	// SourceSelector defines which sources can send data to this destination.
	// If not specified, defaults to "all".
	// +optional
	SourceSelector *SourceSelector `json:"sourceSelector,omitempty"`
}

// DestinationStatus defines the observed state of Destination
type DestinationStatus struct {
	// Represents the observations of a destination's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// Destination is the Schema for the destinations API
type Destination struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DestinationSpec   `json:"spec,omitempty"`
	Status DestinationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DestinationList contains a list of Destination
type DestinationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Destination `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Destination{}, &DestinationList{})
}

var _ config.ExporterConfigurer = &Destination{}

/* Implement common.ExporterConfigurer */
func (dest Destination) GetID() string {
	return dest.Name
}
func (dest Destination) GetType() common.DestinationType {
	return dest.Spec.Type
}
func (dest Destination) GetConfig() map[string]string {
	return dest.Spec.Data
}
func (dest Destination) GetSignals() []common.ObservabilitySignal {
	return dest.Spec.Signals
}

type SourceSelector struct {
	// If a namespace is specified, all workloads (sources) within that namespace are allowed to send data.
	// Example:
	// namespaces: ["default", "production"]
	// This means the destination will receive data from all sources in "default" and "production" namespaces.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
	// Workloads (sources) are assigned to Datastreams via labels (odigos.io/data-stream: true), allowing a more flexible selection mechanism.
	// Example:
	// dataStreams: ["backend", "monitoring"]
	// This means the destination will receive data only from sources labeled with "backend" or "monitoring".
	// +optional
	DataStreams []string `json:"dataStreams,omitempty"`

	// Selection Semantics:
	// If both `Namespaces` and `Groups` are specified, the selection follows an **OR** logic:
	// - A source is included **if** it belongs to **at least one** of the specified namespaces OR groups.
	// - If `Namespaces` is empty but `Groups` is specified, only sources in those groups are included.
	// - If `Groups` is empty but `Namespaces` is specified, all sources in those namespaces are included.
	// - If SourceSelector is nil, the destination receives data from all sources.
}
