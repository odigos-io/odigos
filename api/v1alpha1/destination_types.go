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

// DestinationSpec defines the desired state of Destination
type DestinationSpec struct {
	Type DestinationType `json:"type"`
	Data DestinationData `json:"data"`
}

//+kubebuilder:validation:Enum=grafana;datadog;honeycomb
type DestinationType string

const (
	GrafanaDestinationType   DestinationType = "grafana"
	DatadogDestinationType   DestinationType = "datadog"
	HoneycombDestinationType DestinationType = "honeycomb"
)

type DestinationData struct {
	Grafana   GrafanaData   `json:"grafana,omitempty"`
	Honeycomb HoneycombData `json:"honeycomb,omitempty"`
}

type GrafanaData struct {
	Url    string `json:"url"`
	User   string `json:"user"`
	ApiKey string `json:"apiKey"`
}

type HoneycombData struct {
	ApiKey string `json:"apiKey"`
}

// DestinationStatus defines the observed state of Destination
type DestinationStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
