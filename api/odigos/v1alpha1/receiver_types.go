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
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// ReceiverSpec defines an OpenTelemetry Collector receiver in the Odigos telemetry pipeline.
type ReceiverSpec struct {

	// type of the receiver (hostmetrics, prometheus, filelog, etc).
	// this field is only the type, not its instance name in the collector configuration yaml.
	Type string `json:"type"`

	// optional suffix appended after a slash to disambiguate multiple instances of the same receiver type.
	// The key used in the generated collector config is "{type}/{receiverName}" when set, or just "{type}" when empty.
	ReceiverName string `json:"receiverName,omitempty"`

	// user can attach notes to the receiver, to document its purpose, usage, etc.
	Notes string `json:"notes,omitempty"`

	// disable is a flag to enable or disable the receiver.
	// if the receiver is disabled, it will not be included in the collector configuration yaml.
	// this allows the user to keep the receiver configuration in the CR, but disable it temporarily.
	Disabled bool `json:"disabled,omitempty"`

	// signals controls which observability signal pipelines this receiver is injected into.
	Signals []common.ObservabilitySignal `json:"signals"`

	// collectorRoles controls which collector roles in the Odigos pipeline this receiver is attached to.
	CollectorRoles []CollectorsGroupRole `json:"collectorRoles"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Type=object
	// ReceiverConfig is the configuration of the OpenTelemetry collector receiver component.
	ReceiverConfig runtime.RawExtension `json:"receiverConfig"`
}

// ReceiverStatus defines the observed state of the Receiver.
type ReceiverStatus struct {
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// Receiver is the Schema for an OpenTelemetry Collector Receiver that is added to the Odigos pipeline.
type Receiver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReceiverSpec   `json:"spec,omitempty"`
	Status ReceiverStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ReceiverList contains a list of Receivers.
type ReceiverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Receiver `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Receiver{}, &ReceiverList{})
}

/* Implement common.ReceiverConfigurer */
func (r Receiver) GetID() string {
	return r.Name
}
func (r Receiver) GetType() string {
	return r.Spec.Type
}
func (r Receiver) GetReceiverName() string {
	return r.Spec.ReceiverName
}
func (r Receiver) GetConfig() (config.GenericMap, error) {
	var receiverConfig map[string]interface{}
	if r.Spec.ReceiverConfig.Raw == nil {
		return config.GenericMap{}, nil
	}
	err := json.Unmarshal(r.Spec.ReceiverConfig.Raw, &receiverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal receiver %s data: %v", r.Name, err)
	}
	return receiverConfig, nil
}
func (r Receiver) GetSignals() []common.ObservabilitySignal {
	return r.Spec.Signals
}
func (r Receiver) IsDisabled() bool {
	return r.Spec.Disabled
}
