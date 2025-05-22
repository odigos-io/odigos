package v__internal

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CollectorsGroupRole k8sconsts.CollectorRole

type CollectorsGroupResourcesSettings struct {
	MinReplicas                *int `json:"minReplicas,omitempty"`
	MaxReplicas                *int `json:"maxReplicas,omitempty"`
	MemoryRequestMiB           int  `json:"memoryRequestMiB"`
	MemoryLimitMiB             int  `json:"memoryLimitMiB"`
	CpuRequestMillicores       int  `json:"cpuRequestMillicores"`
	CpuLimitMillicores         int  `json:"cpuLimitMillicores"`
	MemoryLimiterLimitMiB      int  `json:"memoryLimiterLimitMiB"`
	MemoryLimiterSpikeLimitMiB int  `json:"memoryLimiterSpikeLimitMiB"`
	GomemlimitMiB              int  `json:"gomemlimitMiB"`
}

type CollectorsGroupSpec struct {
	Role                    CollectorsGroupRole              `json:"role"`
	CollectorOwnMetricsPort int32                            `json:"collectorOwnMetricsPort"`
	K8sNodeLogsDirectory    string                           `json:"k8sNodeLogsDirectory,omitempty"`
	ResourcesSettings       CollectorsGroupResourcesSettings `json:"resourcesSettings"`
}

type CollectorsGroupStatus struct {
	Ready           bool                         `json:"ready,omitempty"`
	ReceiverSignals []common.ObservabilitySignal `json:"receiverSignals,omitempty"`
	Conditions      []metav1.Condition           `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +genclient
// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type CollectorsGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CollectorsGroupSpec   `json:"spec,omitempty"`
	Status            CollectorsGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type CollectorsGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CollectorsGroup `json:"items"`
}

func (*CollectorsGroup) Hub() {}
