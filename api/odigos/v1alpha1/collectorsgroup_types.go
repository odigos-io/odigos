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
	MinReplicas *int `json:"minReplicas,omitempty"`
	MaxReplicas *int `json:"maxReplicas,omitempty"`

	// MemoryRequestMiB is the memory resource request to be used on the pod template.
	// it will be embedded in the as a resource request of the form `memory: <value>Mi`
	MemoryRequestMiB int `json:"memoryRequestMiB"`

	// This option sets the limit on the memory usage of the collector.
	// since the memory limiter mechanism is heuristic, and operates on fixed intervals,
	// while it cannot fully prevent OOMs, it can help in reducing the chances of OOMs in edge cases.
	// the settings should prevent the collector from exceeding the memory request,
	// so one can set this to the same value as the memory request or higher to allow for some buffer for bursts.
	MemoryLimitMiB int `json:"memoryLimitMiB"`

	// CPU resource request to be used on the pod template.
	// it will be embedded in the as a resource request of the form `cpu: <value>m`
	CpuRequestMillicores int `json:"cpuRequestMillicores"`
	// CPU resource limit to be used on the pod template.
	// it will be embedded in the as a resource limit of the form `cpu: <value>m`
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

type ServiceGraphSettings struct {
	// here so we can add service graph settings in the future without breaking backwards compatibility
}

// configuration for collecting and exporting odigos own metrics.
// e.g. metrics about odigos components, and not about the user's application.
type OdigosOwnMetricsSettings struct {

	// if true, odigos will send all the metrics it collects about itself to the metrics pipeline,
	// which will make them available to the metrics destinations.
	// users can troubleshoot odigos itself by monitoring these metrics in their existing systems,
	// and create their own dashboards, alerting, and more.
	SendToMetricsDestinations bool `json:"sendToMetricsDestinations,omitempty"`

	// if true, odigos will send all the metrics it collects about itself to the odigos metrics store,
	// which is available to odigos UI.
	// it can help in presenting a consistent view of odigos itself, without relying on user system and integrations.
	SendToOdigosMetricsStore bool `json:"sendToOdigosMetricsStore,omitempty"`

	// time interval for flusing metrics (format: 15s, 1m etc). defaults: 10s
	Interval string `json:"interval,omitempty"`
}

type AgentsTelemetrySettings struct {
	// here so we can add agents telemetry settings in the future without breaking backwards compatibility
	// since the collector receives these data points in push mode, and does not record or collect them itself,
	// it is not expected to have many or any settings here.
}

type CollectorsGroupMetricsCollectionSettings struct {

	// if not nil for node collector, it means span to metrics is enabled,
	// and the node collector should set it up in the pipeline.
	// span to metrics is the ability to calculate metrics like http requests/errors/duration etc
	// from the individual spans recorded for relevant operation.
	SpanMetrics *common.MetricsSourceSpanMetricsConfiguration `json:"spanMetrics,omitempty"`

	// if not nil for node collector, it means host metrics is enabled,
	// and the opentelemetry collector "hostmetrics" receiver should be included in the pipeline.
	// host metrics are metrics that are collected from the host node,
	// such as cpu, memory, disk, network, etc.
	HostMetrics *common.MetricsSourceHostMetricsConfiguration `json:"hostMetrics,omitempty"`

	// if not nil for node collector, it means kubelet stats is enabled,
	// and the opentelemetry collector "kubeletstats" receiver should be included in the pipeline.
	// kubelet stats are metrics that are collected from the kubelet point of view,
	// such as cpu, memory, disk, network, per pod, node and more.
	KubeletStats *common.MetricsSourceKubeletStatsConfiguration `json:"kubeletStats,omitempty"`

	// if not nil for cluster collector, it means service graph is enabled,
	// and metrics for the "connectivity" between services should be calculated
	// to be exported to metrics destinations.
	ServiceGraph *ServiceGraphSettings `json:"serviceGraph,omitempty"`

	// if not nil for node collector, it means that some metric destinations are
	// intresseted in collecting metrics about: odigos, the collected data, and the pipeline itself.
	// this allows for users to monitor and operate odigos within their existing system,
	// create dashboards, alerting, and more.
	OdigosOwnMetrics *OdigosOwnMetricsSettings `json:"odigosOwnMetrics,omitempty"`

	// this part controls the metrics which are received from agents in the otlp receiver.
	// it is generally enabled when we want to record metrics, and listed here for completeness.
	// any "otlp receiver" specific settings can go here
	AgentsTelemetry *AgentsTelemetrySettings `json:"agentsTelemetry,omitempty"`
}

// CollectorsGroupSpec defines the desired state of Collector
type CollectorsGroupSpec struct {
	Role CollectorsGroupRole `json:"role"`

	// The port to use for exposing the collector's own metrics as a prometheus endpoint.
	// This can be used to resolve conflicting ports when a collector is using the host network.
	CollectorOwnMetricsPort int32 `json:"collectorOwnMetricsPort"`

	// Additional directory to mount in the node collector pod for logs.
	// This is used to allow the collector to read logs from the host node if /var/log is  symlinked to another directory.
	K8sNodeLogsDirectory string `json:"k8sNodeLogsDirectory,omitempty"`

	// Resources [memory/cpu] settings for the collectors group.
	// these settings are used to protect the collectors instances from:
	// - running out of memory and being killed by the k8s OOM killer
	// - consuming all available memory on the node which can lead to node instability
	// - pushing back pressure to the instrumented applications
	ResourcesSettings CollectorsGroupResourcesSettings `json:"resourcesSettings"`

	// ServiceGraphEnabled is a feature that allows you to visualize the service graph of your application.
	// It is enabled by default and can be disabled by setting the enabled flag to false.
	ServiceGraphDisabled *bool `json:"serviceGraphDisabled,omitempty"`

	// Deprecated - use OtlpExporterConfiguration instead.
	// EnableDataCompression is a feature that allows you to enable data compression before sending data to the Gateway collector.
	// It is disabled by default and can be enabled by setting the enabled flag to true.
	EnableDataCompression *bool `json:"enableDataCompression,omitempty"`

	// OtlpExporterConfiguration is the configuration for the OTLP exporter from node collector to cluster gateway collector.
	OtlpExporterConfiguration *common.OtlpExporterConfiguration `json:"otlpExporterConfiguration,omitempty"`

	// ClusterMetricsEnabled is a feature that allows you to enable the cluster metrics.
	// It is disabled by default and can be enabled by setting the enabled flag to true.
	ClusterMetricsEnabled *bool `json:"clusterMetricsEnabled,omitempty"`

	// for destinations that uses https for exporting data, this value can be used to set the address for an https proxy.
	// when unset or empty, no proxy will be used.
	HttpsProxyAddress *string `json:"httpsProxyAddress,omitempty"`

	// configuration for metrics handling in this collectors group
	// if metric collection is disabled, this will be nil
	// the content is populated only if relevant to this collectors group
	// it's being calculated based on active destinations and their settings,
	// and global settings provided in the odigos configuration or instrumentation rules.
	// it allows for the collector group reconciler to be simplified,
	// and for visibility into the aggregated settings being used to derive configurations deployments and rollouts.
	Metrics *CollectorsGroupMetricsCollectionSettings `json:"metrics,omitempty"`

	// Node selector for the collectors group deployment.
	// Use this to force the gateway to run only on nodes with specific labels.
	// This is a hard requirement: the pod will be scheduled ONLY on nodes that match all labels.
	// If no matching nodes exist, the pod will remain Pending.
	NodeSelector *map[string]string `json:"nodeSelector,omitempty"`

	// Deployment name for the collectors group deployment.
	// Only relevant for cluster gateway collector.
	DeploymentName string `json:"deploymentName,omitempty"`
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
