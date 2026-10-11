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
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ActionNameK8sAttributes = "K8sAttributes"

// +kubebuilder:validation:Enum=pod;namespace;node
type K8sAttributeSource string

const (
	PodAttributeSource       K8sAttributeSource = "pod"
	NamespaceAttributeSource K8sAttributeSource = "namespace"
	NodeAttributeSource      K8sAttributeSource = "node"
)

// K8sAttributeSourcePrecedence defines the precedence order for attribute sources.
// Lower index = lower precedence, higher index = higher precedence.
// When extracting the same label from multiple sources, the source with higher precedence wins.
var K8sAttributeSourcePrecedence = []K8sAttributeSource{
	NodeAttributeSource,      // Lowest precedence
	NamespaceAttributeSource, // Medium precedence
	PodAttributeSource,       // Highest precedence
}

type K8sLabelAttribute struct {
	// The label name to be extracted.
	// e.g. "app.kubernetes.io/name"
	// +kubebuilder:validation:Required
	LabelKey string `json:"labelKey"`
	// The attribute key to be used for the resource attribute created from the label.
	// e.g. "app.kubernetes.name"
	// +kubebuilder:validation:Required
	AttributeKey string `json:"attributeKey"`
	// The source of the label.
	// e.g. "pod" or "namespace"
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=pod
	// Deprecated: Use FromSources instead for specifying multiple sources with precedence.
	From *K8sAttributeSource `json:"from,omitempty"`
	// The sources from which to extract the label, in order of precedence (most specific first).
	// When multiple sources are specified, the most specific source (e.g., pod) takes precedence
	// over less specific sources (e.g., namespace).
	// If a label exists in multiple sources, the value from the most specific source will be used.
	// Supported sources: "pod", "namespace", "node"
	// +kubebuilder:validation:Optional
	FromSources []K8sAttributeSource `json:"fromSources,omitempty"`
}

type K8sAnnotationAttribute struct {
	// The annotation name to be extracted.
	// e.g. "kubectl.kubernetes.io/restartedAt"
	// +kubebuilder:validation:Required
	AnnotationKey string `json:"annotationKey"`
	// The attribute key to be used for the resource attribute created from the annotation.
	// e.g. "kubectl.kubernetes.restartedAt"
	// +kubebuilder:validation:Required
	AttributeKey string `json:"attributeKey"`
	// The source of the annotation.
	// e.g. "pod" or "namespace"
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=pod
	// Deprecated: Use FromSources instead for specifying multiple sources with precedence.
	From *string `json:"from,omitempty"`
	// The sources from which to extract the annotation, in order of precedence (most specific first).
	// When multiple sources are specified, the most specific source (e.g., pod) takes precedence
	// over less specific sources (e.g., namespace).
	// If an annotation exists in multiple sources, the value from the most specific source will be used.
	// Supported sources: "pod", "namespace", "node"
	// +kubebuilder:validation:Optional
	FromSources []K8sAttributeSource `json:"fromSources,omitempty"`
}

type K8sAttributesConfig struct {
	// Collect the following container related attributes:
	// k8s.container.name
	// container.id
	// container.image.name
	// container.image.tag
	CollectContainerAttributes bool `json:"collectContainerAttributes,omitempty"`

	// collect replicaset related attributes (when relevant, e.g. for deployments):
	// k8s.replicaset.name
	// if CollectWorkloadUID is set, also collect:
	// k8s.replicaset.uid
	// DEPRECATED: ReplicaSet attributes are now collected by default during instrumentation.
	CollectReplicaSetAttributes bool `json:"collectReplicaSetAttributes,omitempty"`

	// Collect the following workload UID attributes:
	// k8s.deployment.uid
	// k8s.daemonset.uid
	// k8s.statefulset.uid
	// DEPRECATED: Workload UID attributes are now collected by default during instrumentation.
	CollectWorkloadUID bool `json:"collectWorkloadUID,omitempty"`

	// Collect the k8s.cluster.uid attribute, which is set to the uid of the namespace "kube-system"
	CollectClusterUID bool `json:"collectClusterUID,omitempty"`

	// list of labels to be extracted from the pod, and the attribute key to be used for the resource attribute created from each label.
	// +optional
	LabelsAttributes []K8sLabelAttribute `json:"labelsAttributes,omitempty"`

	// list of annotations to be extracted from the pod, and the attribute key to be used for the resource attribute created from each annotation.
	// +optional
	AnnotationsAttributes []K8sAnnotationAttribute `json:"annotationsAttributes,omitempty"`
}

func (K8sAttributesConfig) ProcessorType() string {
	return "k8sattributes"
}

func (K8sAttributesConfig) OrderHint() int {
	return 0
}

func (K8sAttributesConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleNodeCollector,
	}
}

type K8sAttributesSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	// Collect the following container related attributes:
	// k8s.container.name
	// container.id
	// container.image.name
	// container.image.tag
	CollectContainerAttributes bool `json:"collectContainerAttributes,omitempty"`

	// collect replicaset related attributes (when relevant, e.g. for deployments):
	// k8s.replicaset.name
	// if CollectWorkloadUID is set, also collect:
	// k8s.replicaset.uid
	// DEPRECATED: ReplicaSet attributes are now collected by default during instrumentation.
	CollectReplicaSetAttributes bool `json:"collectReplicaSetAttributes,omitempty"`

	// Collect the following workload UID attributes:
	// k8s.deployment.uid
	// k8s.daemonset.uid
	// k8s.statefulset.uid
	// DEPRECATED: Workload UID attributes are now collected by default during instrumentation.
	CollectWorkloadUID bool `json:"collectWorkloadUID,omitempty"`

	// Collect the k8s.cluster.uid attribute, which is set to the uid of the namespace "kube-system"
	CollectClusterUID bool `json:"collectClusterUID,omitempty"`

	// list of labels to be extracted from the pod, and the attribute key to be used for the resource attribute created from each label.
	// +optional
	LabelsAttributes []K8sLabelAttribute `json:"labelsAttributes,omitempty"`

	// list of annotations to be extracted from the pod, and the attribute key to be used for the resource attribute created from each annotation.
	// +optional
	AnnotationsAttributes []K8sAnnotationAttribute `json:"annotationsAttributes,omitempty"`
}

// K8sAttributesStatus defines the observed state of K8sAttributes action
type K8sAttributesStatus struct {
	// Represents the observations of a k8sattributes' current state.
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
//+kubebuilder:resource:path=k8sattributesresolvers,scope=Namespaced
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// K8sAttributesResolver allows adding an action to collect k8s attributes.
// DEPRECATED: Use odigosv1.Action instead
type K8sAttributesResolver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sAttributesSpec   `json:"spec,omitempty"`
	Status K8sAttributesStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sAttributesResolverList contains a list of K8sAttributes
// DEPRECATED: Use odigosv1.ActionList instead
type K8sAttributesResolverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sAttributesResolver `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sAttributesResolver{}, &K8sAttributesResolverList{})
}
