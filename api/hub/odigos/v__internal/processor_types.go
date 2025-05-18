package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type ProcessorSpec struct {
	Type            string                       `json:"type"`
	ProcessorName   string                       `json:"processorName,omitempty"`
	Notes           string                       `json:"notes,omitempty"`
	Disabled        bool                         `json:"disabled,omitempty"`
	Signals         []common.ObservabilitySignal `json:"signals"`
	CollectorRoles  []CollectorsGroupRole        `json:"collectorRoles"`
	OrderHint       int                          `json:"orderHint,omitempty"`
	ProcessorConfig runtime.RawExtension         `json:"processorConfig"`
}

type ProcessorStatus struct {
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type Processor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProcessorSpec   `json:"spec,omitempty"`
	Status            ProcessorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ProcessorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Processor `json:"items"`
}

func (*Processor) Hub() {}
