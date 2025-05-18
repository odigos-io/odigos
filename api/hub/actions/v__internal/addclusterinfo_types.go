package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OtelAttributeWithValue struct {
	AttributeName        string  `json:"attributeName"`
	AttributeStringValue *string `json:"attributeStringValue"`
}

const ActionNameAddClusterInfo = "AddClusterInfo"

type AddClusterInfoConfig struct {
	ClusterAttributes []OtelAttributeWithValue `json:"clusterAttributes"`
}

type AddClusterInfoSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	ClusterAttributes []OtelAttributeWithValue `json:"clusterAttributes"`
}

type AddClusterInfoStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type AddClusterInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddClusterInfoSpec   `json:"spec,omitempty"`
	Status AddClusterInfoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AddClusterInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AddClusterInfo `json:"items"`
}

func (*AddClusterInfo) Hub() {}
