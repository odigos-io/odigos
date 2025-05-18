package v__internal

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SpanAttributeSamplerSpec struct {
	ActionName       string                       `json:"actionName,omitempty"`
	Notes            string                       `json:"notes,omitempty"`
	Disabled         bool                         `json:"disabled,omitempty"`
	Signals          []common.ObservabilitySignal `json:"signals"`
	AttributeFilters []SpanAttributeFilter        `json:"attribute_filters"`
}

type SpanAttributeFilter struct {
	ServiceName           string             `json:"service_name"`
	AttributeKey          string             `json:"attribute_key"`
	Condition             AttributeCondition `json:"condition"`
	SamplingRatio         float64            `json:"sampling_ratio"`
	FallbackSamplingRatio float64            `json:"fallback_sampling_ratio"`
}

type AttributeCondition struct {
	StringCondition  *StringAttributeCondition  `json:"string_condition,omitempty"`
	NumberCondition  *NumberAttributeCondition  `json:"number_condition,omitempty"`
	BooleanCondition *BooleanAttributeCondition `json:"boolean_condition,omitempty"`
	JsonCondition    *JsonAttributeCondition    `json:"json_condition,omitempty"`
}

type StringAttributeCondition struct {
	Operation     string `json:"operation"`
	ExpectedValue string `json:"expected_value,omitempty"`
}

type JsonAttributeCondition struct {
	Operation     string `json:"operation"`
	JsonPath      string `json:"json_path,omitempty"`
	ExpectedValue string `json:"expected_value,omitempty"`
}

type NumberAttributeCondition struct {
	Operation     string  `json:"operation"`
	ExpectedValue float64 `json:"expected_value,omitempty"`
}

type BooleanAttributeCondition struct {
	Operation     string `json:"operation"`
	ExpectedValue bool   `json:"expected_value,omitempty"`
}

type SpanAttributeSamplerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type SpanAttributeSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpanAttributeSamplerSpec   `json:"spec,omitempty"`
	Status SpanAttributeSamplerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type SpanAttributeSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpanAttributeSampler `json:"items"`
}

func (*SpanAttributeSampler) Hub() {}
