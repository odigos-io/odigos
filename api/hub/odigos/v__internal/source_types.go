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

package v__internal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// Source configures an application for auto-instrumentation.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels=odigos.io/system-object=true
// +kubebuilder:skipversion
// +kubebuilder:storageversion
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

// +kubebuilder:object:root=true
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

// Hub marks this as canonical.
func (*Source) Hub() {}
