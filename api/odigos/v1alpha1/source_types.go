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
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

var ErrorTooManySources = errors.New("too many Sources found for workload")

// Source configures an application for auto-instrumentation.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Workload",type=string,JSONPath=`.spec.workload.name`
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.workload.kind`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.workload.namespace`
type Source struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SourceSpec   `json:"spec"`
	Status SourceStatus `json:"status,omitempty"`
}

type SourceSpec struct {
	// Workload represents the workload or namespace to be instrumented.
	// This field is required upon creation and cannot be modified.
	// +kubebuilder:validation:Required
	Workload workload.PodWorkload `json:"workload"`
	// DisableInstrumentation excludes this workload from auto-instrumentation.
	// +kubebuilder:validation:Optional
	DisableInstrumentation bool `json:"disableInstrumentation,omitempty"`
}

type SourceStatus struct {
	// Represents the observations of a source's current state.
	// Known .status.conditions.type are: "Available", "Progressing"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true

type SourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Source `json:"items"`
}

// +kubebuilder:object:generate=false

type WorkloadSources struct {
	Workload  *Source
	Namespace *Source
}

// GetSources returns a WorkloadSources listing the Workload and Namespace Source
// that currently apply to the given object. In theory, this should only ever return at most
// 1 Namespace and/or 1 Workload Source for an object. If more are found, an error is returned.
func GetSources(ctx context.Context, kubeClient client.Client, obj client.Object) (*WorkloadSources, error) {
	var err error
	workloadSources := &WorkloadSources{}

	namespace := obj.GetNamespace()
	if len(namespace) == 0 && obj.GetObjectKind().GroupVersionKind().Kind == "Namespace" {
		namespace = obj.GetName()
	}

	if obj.GetObjectKind().GroupVersionKind().Kind != "Namespace" {
		sourceList := SourceList{}
		selector := labels.SelectorFromSet(labels.Set{
			consts.WorkloadNameLabel:      obj.GetName(),
			consts.WorkloadNamespaceLabel: namespace,
			consts.WorkloadKindLabel:      obj.GetObjectKind().GroupVersionKind().Kind,
		})
		err := kubeClient.List(ctx, &sourceList, &client.ListOptions{LabelSelector: selector}, client.InNamespace(namespace))
		if err != nil {
			return nil, err
		}
		if len(sourceList.Items) > 1 {
			return nil, ErrorTooManySources
		}
		if len(sourceList.Items) == 1 {
			workloadSources.Workload = &sourceList.Items[0]
		}
	}

	namespaceSourceList := SourceList{}
	namespaceSelector := labels.SelectorFromSet(labels.Set{
		consts.WorkloadNameLabel:      namespace,
		consts.WorkloadNamespaceLabel: namespace,
		consts.WorkloadKindLabel:      "Namespace",
	})
	err = kubeClient.List(ctx, &namespaceSourceList, &client.ListOptions{LabelSelector: namespaceSelector}, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}
	if len(namespaceSourceList.Items) > 1 {
		return nil, ErrorTooManySources
	}
	if len(namespaceSourceList.Items) == 1 {
		workloadSources.Namespace = &namespaceSourceList.Items[0]
	}

	return workloadSources, nil
}

// IsExcludedSource returns true if the Source is disabling instrumentation.
func IsExcludedSource(source *Source) bool {
	return source.Spec.DisableInstrumentation
}

func init() {
	SchemeBuilder.Register(&Source{}, &SourceList{})
}
