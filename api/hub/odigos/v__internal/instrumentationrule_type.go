package v__internal

import (
	"github.com/odigos-io/odigos/api/hub/odigos/v__internal/instrumentationrules"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstrumentationLibraryGlobalId struct {
	Name     string                     `json:"name"`
	SpanKind common.SpanKind            `json:"spanKind,omitempty"`
	Language common.ProgrammingLanguage `json:"language"`
}

type InstrumentationRuleSpec struct {
	RuleName                 string                                      `json:"ruleName,omitempty"`
	Notes                    string                                      `json:"notes,omitempty"`
	Disabled                 bool                                        `json:"disabled,omitempty"`
	Workloads                *[]k8sconsts.PodWorkload                    `json:"workloads,omitempty"`
	InstrumentationLibraries *[]InstrumentationLibraryGlobalId           `json:"instrumentationLibraries,omitempty"`
	PayloadCollection        *instrumentationrules.PayloadCollection     `json:"payloadCollection,omitempty"`
	OtelSdks                 *instrumentationrules.OtelSdks              `json:"otelSdks,omitempty"`
	OtelDistros              *instrumentationrules.OtelDistros           `json:"otelDistros,omitempty"`
	CodeAttributes           *instrumentationrules.CodeAttributes        `json:"codeAttributes,omitempty"`
	HeadersCollection        *instrumentationrules.HttpHeadersCollection `json:"headersCollection,omitempty"`
}

type InstrumentationRuleStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type InstrumentationRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              InstrumentationRuleSpec   `json:"spec,omitempty"`
	Status            InstrumentationRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type InstrumentationRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentationRule `json:"items"`
}

func (*InstrumentationRule) Hub() {}
