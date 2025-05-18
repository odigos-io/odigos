package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8sLabelAttribute struct {
	LabelKey     string `json:"labelKey"`
	AttributeKey string `json:"attributeKey"`
}

type K8sAnnotationAttribute struct {
	AnnotationKey string `json:"annotationKey"`
	AttributeKey  string `json:"attributeKey"`
}

type K8sAttributesSpec struct {
	ActionName                  string                       `json:"actionName,omitempty"`
	Notes                       string                       `json:"notes,omitempty"`
	Disabled                    bool                         `json:"disabled,omitempty"`
	Signals                     []common.ObservabilitySignal `json:"signals"`
	CollectContainerAttributes  bool                         `json:"collectContainerAttributes,omitempty"`
	CollectReplicaSetAttributes bool                         `json:"collectReplicaSetAttributes,omitempty"`
	CollectWorkloadUID          bool                         `json:"collectWorkloadUID,omitempty"`
	CollectClusterUID           bool                         `json:"collectClusterUID,omitempty"`
	LabelsAttributes            []K8sLabelAttribute          `json:"labelsAttributes,omitempty"`
	AnnotationsAttributes       []K8sAnnotationAttribute     `json:"annotationsAttributes,omitempty"`
}

type K8sAttributesStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type K8sAttributesResolver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sAttributesSpec   `json:"spec,omitempty"`
	Status K8sAttributesStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type K8sAttributesResolverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sAttributesResolver `json:"items"`
}

func (*K8sAttributesResolver) Hub() {}
