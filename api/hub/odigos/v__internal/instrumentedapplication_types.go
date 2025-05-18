package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigOption struct {
	OptionKey string          `json:"optionKey"`
	SpanKind  common.SpanKind `json:"spanKind"`
}

type InstrumentationLibraryOptions struct {
	LibraryName string         `json:"libraryName"`
	Options     []ConfigOption `json:"options"`
}

type OptionByContainer struct {
	ContainerName            string                          `json:"containerName"`
	InstrumentationLibraries []InstrumentationLibraryOptions `json:"instrumentationsLibraries"`
}

type InstrumentedApplicationSpec struct {
	RuntimeDetails []RuntimeDetailsByContainer `json:"runtimeDetails,omitempty"`
	Options        []OptionByContainer         `json:"options,omitempty"`
}

type InstrumentedApplicationStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type InstrumentedApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentedApplicationSpec   `json:"spec,omitempty"`
	Status InstrumentedApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type InstrumentedApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentedApplication `json:"items"`
}

func (*InstrumentedApplication) Hub() {}
