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
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/common"

	actions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ActionMigratedLegacyPrefix = "migrated-legacy-"

// condition types for action CR
const (
	// TransformedToProcessor is the condition when the action CR is transformed to a processor CR.
	// This is the first step in the reconciliation process.
	ActionTransformedToProcessorType = "TransformedToProcessor"
)

// +kubebuilder:validation:Enum=ProcessorCreated;FailedToCreateProcessor;FailedToTransformToProcessor;ProcessorNotRequired
type ActionTransformedToProcessorReason string

// Reasons for action condition types
const (
	// ProcessorCreatedReason is added to the action when the processor CR is created.
	ActionTransformedToProcessorReasonProcessorCreated ActionTransformedToProcessorReason = "ProcessorCreated"
	// FailedToCreateProcessorReason is added to the action when the processor CR creation fails.
	ActionTransformedToProcessorReasonFailedToCreateProcessor ActionTransformedToProcessorReason = "FailedToCreateProcessor"
	// FailedToTransformToProcessorReason is added to the action when the transformation to processor object fails.
	ActionTransformedToProcessorReasonFailedToTransformToProcessorReason ActionTransformedToProcessorReason = "FailedToTransformToProcessor"
	// ActionTransformedToProcessorReasonProcessorNotRequired is added to the action when the action should not need to be transformed to a processor CR. e.g. when the action is a URL templatization action.
	ActionTransformedToProcessorReasonProcessorNotRequired ActionTransformedToProcessorReason = "ProcessorNotRequired"
)

type ActionSpec struct {
	// Allows you to attach a meaningful name to the action for convenience. Odigos does not use or assume any meaning from this field.
	ActionName string `json:"actionName,omitempty"`

	// A free-form text field that allows you to attach notes regarding the action for convenience. For example: why it was added. Odigos does not use or assume any meaning from this field.
	Notes string `json:"notes,omitempty"`

	// A boolean field allowing to temporarily disable the action, but keep it around for future use
	Disabled bool `json:"disabled,omitempty"`

	// Which signals should this action operate on.
	Signals []common.ObservabilitySignal `json:"signals"`

	// AddClusterInfo is the config for the AddClusterInfo Action.
	AddClusterInfo *actionsv1.AddClusterInfoConfig `json:"addClusterInfo,omitempty"`

	// DeleteAttribute is the config for the DeleteAttribute Action.
	DeleteAttribute *actionsv1.DeleteAttributeConfig `json:"deleteAttribute,omitempty"`

	// RenameAttribute is the config for the RenameAttribute Action.
	RenameAttribute *actionsv1.RenameAttributeConfig `json:"renameAttribute,omitempty"`

	// PiiMasking is the config for the PiiMasking Action.
	PiiMasking *actionsv1.PiiMaskingConfig `json:"piiMasking,omitempty"`

	// K8sAttributes is the config for the K8sAttributes Action.
	K8sAttributes *actionsv1.K8sAttributesConfig `json:"k8sAttributes,omitempty"`

	// Samplers is the config for the Samplers Action.
	Samplers *actionsv1.SamplersConfig `json:"samplers,omitempty"`

	// URLTemplatization is the config for the URLTemplatization Action.
	URLTemplatization *actions.URLTemplatizationConfig `json:"urlTemplatization,omitempty"`
}

type ActionStatus struct {
	// Represents the observations of a action's current state.
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
//+kubebuilder:metadata:labels=odigos.io/system-object=true

type Action struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionSpec   `json:"spec,omitempty"`
	Status ActionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type ActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Action `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Action{}, &ActionList{})
}
