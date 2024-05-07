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

//+kubebuilder:object:generate=true
type ConfigOption struct {
	OptionKey string `json:"optionKey"`
	SpanKind  common.SpanKind `json:"spanKind"`
}

//+kubebuilder:object:generate=true
type InstrumentationLibraryOptions struct {
	LibraryName string `json:"libraryName"`
	Options []ConfigOption `json:"options"`
}

//+kubebuilder:object:generate=true
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

//+kubebuilder:object:generate=true
type RuntimeDetailsByContainer struct {
	ContainerName string              `json:"containerName"`
	Language      common.ProgrammingLanguage `json:"language"`
	EnvVars       []EnvVar            `json:"envVars,omitempty"`
}

// +kubebuilder:object:generate=true
type OptionByContainer struct {
	ContainerName string `json:"containerName"`
	InstrumentationLibraries []InstrumentationLibraryOptions `json:"instrumentationsLibraries"`
}

// InstrumentedApplicationSpec defines the desired state of InstrumentedApplication
type InstrumentedApplicationSpec struct {
	RuntimeDetails []RuntimeDetailsByContainer `json:"runtimeDetails,omitempty"`
	Options  []OptionByContainer   `json:"options,omitempty"`
}

// InstrumentedApplicationStatus defines the observed state of InstrumentedApplication
type InstrumentedApplicationStatus struct {
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// InstrumentedApplication is the Schema for the instrumentedapplications API
type InstrumentedApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentedApplicationSpec   `json:"spec,omitempty"`
	Status InstrumentedApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InstrumentedApplicationList contains a list of InstrumentedApplication
type InstrumentedApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentedApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstrumentedApplication{}, &InstrumentedApplicationList{})
}
