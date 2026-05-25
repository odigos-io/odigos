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
	"regexp"

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

type ContainerOverride struct {
	// The name of the container to override.
	ContainerName string `json:"containerName"`

	// RuntimeInfo to use for agent enabling.
	// If set for a container, the automatic detection will not be used for this container,
	// and the distro to use will be calculated based on this value.
	RuntimeInfo *RuntimeDetailsByContainer `json:"runtimeInfo,omitempty"`

	// select one specific otel distro to use for this container.
	// the selected distro must match the programming language and any constraints of the distro.
	// if the programmin language is incorrect, the runtime info can be overridden to match
	OtelDistroName *string `json:"otelDistroName,omitempty"`
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

	// Specify specific override values for containers in a workload source.
	// Not valid for namespace sources.
	// Can be used to set the runtime info in case the automatic detection fails or produce wrong results.
	// Containers are identified by their names.
	// All containers not listed will retain their default behavior.
	// +kubebuilder:validation:Optional
	// +optional
	ContainerOverrides []ContainerOverride `json:"containerOverrides,omitempty"`

	// MatchWorkloadNameAsRegex indicates that the workload name should be matched using regex.
	// When true, spec.workload.name is treated as a regular expression pattern.
	// This allows matching multiple workloads with a single Source CRD.
	// Not valid for namespace sources.
	// +kubebuilder:validation:Optional
	// +optional
	MatchWorkloadNameAsRegex bool `json:"matchWorkloadNameAsRegex,omitempty"`
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
func GetSources(ctx context.Context, kubeClient client.Client, pw k8sconsts.PodWorkload) (*WorkloadSources, error) {
	var err error
	workloadSources := &WorkloadSources{}

	namespace := pw.Namespace
	if len(namespace) == 0 && pw.Kind == k8sconsts.WorkloadKindNamespace {
		namespace = pw.Name
	}

	if pw.Kind != k8sconsts.WorkloadKindNamespace {
		sourceList := SourceList{}

		// First, list all sources with matching namespace and kind
		// We need to check both exact matches and regex matches
		selector := labels.SelectorFromSet(labels.Set{
			k8sconsts.WorkloadNamespaceLabel: namespace,
			k8sconsts.WorkloadKindLabel:      string(pw.Kind),
		})
		err := kubeClient.List(ctx, &sourceList, &client.ListOptions{LabelSelector: selector}, client.InNamespace(namespace))
		if err != nil {
			return nil, err
		}

		// Filter sources: exact match or regex match
		var matchingSources []Source
		for _, source := range sourceList.Items {
			if source.Spec.MatchWorkloadNameAsRegex {
				// Compile and match regex pattern
				pattern := source.Spec.Workload.Name
				matched, err := regexp.MatchString(pattern, pw.Name)
				if err != nil {
					// Invalid regex pattern, skip this source
					continue
				}
				if matched {
					matchingSources = append(matchingSources, source)
				}
			} else {
				// Exact match
				if source.Spec.Workload.Name == pw.Name {
					matchingSources = append(matchingSources, source)
				}
			}
		}

		if len(matchingSources) > 1 {
			return nil, ErrorTooManySources
		}
		if len(matchingSources) == 1 {
			workloadSources.Workload = &matchingSources[0]
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
