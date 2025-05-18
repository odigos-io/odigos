package v__internal

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DestinationSpec struct {
	Type            common.DestinationType       `json:"type"`
	DestinationName string                       `json:"destinationName"`
	Data            map[string]string            `json:"data"`
	SecretRef       *v1.LocalObjectReference     `json:"secretRef,omitempty"`
	Signals         []common.ObservabilitySignal `json:"signals"`

	SourceSelector *SourceSelector `json:"sourceSelector,omitempty"`
}

type DestinationStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type Destination struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DestinationSpec   `json:"spec,omitempty"`
	Status DestinationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DestinationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Destination `json:"items"`
}

var _ config.ExporterConfigurer = &Destination{}

func (dest Destination) GetID() string {
	return dest.Name
}
func (dest Destination) GetType() common.DestinationType {
	return dest.Spec.Type
}
func (dest Destination) GetConfig() map[string]string {
	return dest.Spec.Data
}
func (dest Destination) GetSignals() []common.ObservabilitySignal {
	return dest.Spec.Signals
}

func (*Destination) Hub() {}
