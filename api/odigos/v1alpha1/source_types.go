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

	"github.com/odigos-io/odigos/api/k8sconsts"
)

var ErrorTooManySources = errors.New("too many Sources found for workload")

// Source configures an application for auto-instrumentation.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Workload",type=string,JSONPath=`.spec.workload.name`
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.workload.kind`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.workload.namespace`
// +kubebuilder:printcolumn:name="Disabled",type=string,JSONPath=`.spec.disableInstrumentation`
// +kubebuilder:metadata:labels=odigos.io/system-object=true
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
	Workload k8sconsts.PodWorkload `json:"workload"`
	// DisableInstrumentation excludes this workload from auto-instrumentation.
	// +kubebuilder:validation:Optional
	DisableInstrumentation bool `json:"disableInstrumentation,omitempty"`
	// OtelServiceName determines the "service.name" resource attribute which will be reported by the instrumentations of this source.
	// If not set, the workload name will be used.
	// It is not valid for namespace sources.
	// +kubebuilder:validation:Optional
	// +optional
	OtelServiceName string `json:"otelServiceName,omitempty"`
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

type SourceSelector struct {
	// If a namespace is specified, all workloads (sources) within that namespace are allowed to send data.
	// Example:
	// namespaces: ["default", "production"]
	// This means the destination will receive data from all sources in "default" and "production" namespaces.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
	// Workloads (sources) are assigned to groups via labels (odigos.io/group-backend: true), allowing a more flexible selection mechanism.
	// Example:
	// groups: ["backend", "monitoring"]
	// This means the destination will receive data only from sources labeled with "backend" or "monitoring".
	// +optional
	Groups []string `json:"groups,omitempty"`

	// Selection Semantics:
	// If both `Namespaces` and `Groups` are specified, the selection follows an **OR** logic:
	// - A source is included **if** it belongs to **at least one** of the specified namespaces OR groups.
	// - If `Namespaces` is empty but `Groups` is specified, only sources in those groups are included.
	// - If `Groups` is empty but `Namespaces` is specified, all sources in those namespaces are included.
	// - If SourceSelector is nil, the destination receives data from all sources.
}

// GetSources returns a WorkloadSources listing the Workload and Namespace Source
// that currently apply to the given object. In theory, this should only ever return at most
// 1 Namespace and/or 1 Workload Source for an object. If more are found, an error is returned.
func GetSources(ctx context.Context, kubeClient client.Client, workloadObj k8sconsts.PodWorkload) (*WorkloadSources, error) {
	var err error
	workloadSources := &WorkloadSources{}

	namespace := workloadObj.Namespace
	if len(namespace) == 0 && workloadObj.Kind == k8sconsts.WorkloadKindNamespace {
		namespace = workloadObj.Name
	}

	if workloadObj.Kind != k8sconsts.WorkloadKindNamespace {
		sourceList := SourceList{}
		selector := labels.SelectorFromSet(labels.Set{
			k8sconsts.WorkloadNameLabel:      workloadObj.Name,
			k8sconsts.WorkloadNamespaceLabel: namespace,
			k8sconsts.WorkloadKindLabel:      string(workloadObj.Kind),
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
		k8sconsts.WorkloadNameLabel:      namespace,
		k8sconsts.WorkloadNamespaceLabel: namespace,
		k8sconsts.WorkloadKindLabel:      string(k8sconsts.WorkloadKindNamespace),
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

// IsDisabledSource returns true if the Source is disabling instrumentation.
func IsDisabledSource(source *Source) bool {
	return source.Spec.DisableInstrumentation
}

func init() {
	SchemeBuilder.Register(&Source{}, &SourceList{})
}
