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

type K8sLabelAttribute struct {
	// The label name to be extracted from the pod.
	// e.g. "app.kubernetes.io/name"
	// +kubebuilder:validation:Required
	LabelKey string `json:"labelKey"`
	// The attribute key to be used for the resource attribute created from the label.
	// e.g. "app.kubernetes.name"
	// +kubebuilder:validation:Required
	AttributeKey string `json:"attributeKey"`
}

type K8sAnnotationAttribute struct {
	// The label name to be extracted from the pod.
	// e.g. "kubectl.kubernetes.io/restartedAt"
	// +kubebuilder:validation:Required
	AnnotationKey string `json:"annotationKey"`
	// The attribute key to be used for the resource attribute created from the label.
	// e.g. "kubectl.kubernetes.restartedAte"
	// +kubebuilder:validation:Required
	AttributeKey string `json:"attributeKey"`
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
	CollectReplicaSetAttributes bool `json:"collectReplicaSetAttributes,omitempty"`

	// Collect the following workload UID attributes:
	// k8s.deployment.uid
	// k8s.daemonset.uid
	// k8s.statefulset.uid
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
type K8sAttributesResolver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sAttributesSpec   `json:"spec,omitempty"`
	Status K8sAttributesStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sAttributesResolverList contains a list of K8sAttributes
type K8sAttributesResolverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sAttributesResolver `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sAttributesResolver{}, &K8sAttributesResolverList{})
}
