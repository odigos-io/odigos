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
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=CLUSTER_GATEWAY;NODE_COLLECTOR
type CollectorsGroupRole k8sconsts.CollectorRole

const (
	CollectorsGroupRoleClusterGateway CollectorsGroupRole = CollectorsGroupRole(k8sconsts.CollectorsRoleClusterGateway)
	CollectorsGroupRoleNodeCollector  CollectorsGroupRole = CollectorsGroupRole(k8sconsts.CollectorsRoleNodeCollector)
)

// The raw values to control the collectors group resources and behavior.
// any defaulting, validations and calculations should be done in the controllers
// that create this CR.
// Values will be used as is without any further processing.
type CollectorsGroupResourcesSettings struct {

	// Minumum + Maximum number of replicas for the collector - these relevant only for gateway.
	MinReplicas *int `json:"minReplicas"`
	MaxReplicas *int `json:"maxReplicas"`

	// MemoryRequestMiB is the memory resource request to be used on the pod template.
	// it will be embedded in the as a resource request of the form "memory: <value>Mi"
	MemoryRequestMiB int `json:"memoryRequestMiB"`

	// This option sets the limit on the memory usage of the collector.
	// since the memory limiter mechanism is heuristic, and operates on fixed intervals,
	// while it cannot fully prevent OOMs, it can help in reducing the chances of OOMs in edge cases.
	// the settings should prevent the collector from exceeding the memory request,
	// so one can set this to the same value as the memory request or higher to allow for some buffer for bursts.
	MemoryLimitMiB int `json:"memoryLimitMiB"`

	// CPU resource request to be used on the pod template.
	// it will be embedded in the as a resource request of the form "cpu: <value>m"
	CpuRequestMillicores int `json:"cpuRequestMillicores"`
	// CPU resource limit to be used on the pod template.
	// it will be embedded in the as a resource limit of the form "cpu: <value>m"
	CpuLimitMillicores int `json:"cpuLimitMillicores"`

	// this parameter sets the "limit_mib" parameter in the memory limiter configuration for the collector.
	// it is the hard limit after which a force garbage collection will be performed.
	// this value will end up comparing against the go runtime reported heap Alloc value.
	// According to the memory limiter docs:
	// > Note that typically the total memory usage of process will be about 50MiB higher than this value
	// a test from nov 2024 showed that fresh odigos collector with no traffic takes 38MiB,
	// thus the 50MiB is a good value to start with.
	MemoryLimiterLimitMiB int `json:"memoryLimiterLimitMiB"`

	// this parameter sets the "spike_limit_mib" parameter in the memory limiter configuration for the collector memory limiter.
	// note that this is not the processor soft limit itself, but the diff in Mib between the hard limit and the soft limit.
	// according to the memory limiter docs, it is recommended to set this to 20% of the hard limit.
	// changing this value allows trade-offs between memory usage and resiliency to bursts.
	MemoryLimiterSpikeLimitMiB int `json:"memoryLimiterSpikeLimitMiB"`

	// the GOMEMLIMIT environment variable value for the collector pod.
	// this is when go runtime will start garbage collection.
	// it is recommended to be set to 80% of the hard limit of the memory limiter.
	GomemlimitMiB int `json:"gomemlimitMiB"`
}

// CollectorsGroupSpec defines the desired state of Collector
type CollectorsGroupSpec struct {
	Role CollectorsGroupRole `json:"role"`

	// The port to use for exposing the collector's own metrics as a prometheus endpoint.
	// This can be used to resolve conflicting ports when a collector is using the host network.
	CollectorOwnMetricsPort int32 `json:"collectorOwnMetricsPort"`

	// Resources [memory/cpu] settings for the collectors group.
	// these settings are used to protect the collectors instances from:
	// - running out of memory and being killed by the k8s OOM killer
	// - consuming all available memory on the node which can lead to node instability
	// - pushing back pressure to the instrumented applications
	ResourcesSettings CollectorsGroupResourcesSettings `json:"resourcesSettings"`
}

// CollectorsGroupStatus defines the observed state of Collector
type CollectorsGroupStatus struct {
	Ready bool `json:"ready,omitempty"`

	// Receiver Signals are the signals (trace, metrics, logs) that the collector has setup
	// an otlp receiver for, thus it can accept data from an upstream component.
	// this is used to determine if a workload should export each signal or not.
	// this list is calculated based on the odigos destinations that were configured
	ReceiverSignals []common.ObservabilitySignal `json:"receiverSignals,omitempty"`

	// Represents the observations of a collectorsroup's current state.
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
//+kubebuilder:metadata:labels=odigos.io/config=1
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// CollectorsGroup is the Schema for the collectors API
type CollectorsGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CollectorsGroupSpec   `json:"spec,omitempty"`
	Status CollectorsGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CollectorsGroupList contains a list of Collector
type CollectorsGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CollectorsGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CollectorsGroup{}, &CollectorsGroupList{})
}
