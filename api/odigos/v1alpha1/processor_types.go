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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// ProcessorSpec defines the an OpenTelemetry Collector processor in odigos telemetry pipeline
type ProcessorSpec struct {

	// type of the processor (batch, attributes, etc).
	// this field is only the type, not it's instance name in the collector configuration yaml
	Type string `json:"type"`

	// this name is solely for the user convenience, to attach a meaningful name to the processor.
	// odigos must not assume any semantics from this name.
	// odigos cannot assume this name is unique, not empty, exclude spaces or dots, limited in length, etc.
	ProcessorName string `json:"processorName,omitempty"`

	// user can attach notes to the processor, to document its purpose, usage, etc.
	Notes string `json:"notes,omitempty"`

	// disable is a flag to enable or disable the processor.
	// if the processor is disabled, it will not be included in the collector configuration yaml.
	// this allows the user to keep the processor configuration in the CR, but disable it temporarily.
	Disabled bool `json:"disabled,omitempty"`

	// signals can be used to control which observability signals are processed by the processor.
	Signals []common.ObservabilitySignal `json:"signals"`

	// control which collector roles in odigos pipeline this processor is attached to.
	CollectorRoles []CollectorsGroupRole `json:"collectorRoles"`

	// control the order of processors.
	// a processor with lower order hint value will be placed before other processors with higher value.
	// if 2 processors have the same value, the order is arbitrary.
	// if the value is missing (or 0) the processor can be placed anywhere in the pipeline
	OrderHint int `json:"orderHint,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	// this it the configuration of the opentelemetry collector processor component with the type specified in 'type'.
	ProcessorConfig runtime.RawExtension `json:"processorConfig"`
}

// ProcessorStatus defines the observed state of the processor
type ProcessorStatus struct {
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Processor is the Schema for an Opentelemetry Collector Processor that is added to Odigos pipeline
type Processor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProcessorSpec   `json:"spec,omitempty"`
	Status ProcessorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProcessorList contains a list of Processors
type ProcessorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Processor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Processor{}, &ProcessorList{})
}
