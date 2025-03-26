package v1alpha1

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpanAttributeSamplerSpec defines the desired state of SpanAttributeSampler
type SpanAttributeSamplerSpec struct {
	ActionName string                       `json:"actionName,omitempty"`
	Notes      string                       `json:"notes,omitempty"`
	Disabled   bool                         `json:"disabled,omitempty"`
	Signals    []common.ObservabilitySignal `json:"signals"`

	// Filters based on span attributes and conditions
	// +kubebuilder:validation:Required
	AttributeFilters []SpanAttributeFilter `json:"attributeFilters"`
}

// SpanAttributeFilter allows sampling traces based on span attributes and conditions.
type SpanAttributeFilter struct {
	// Specifies the service the filter applies to
	// +kubebuilder:validation:Required
	ServiceName string `json:"service_name"`

	// Attribute key to evaluate
	// +kubebuilder:validation:Required
	AttributeKey string `json:"attribute_key"`

	// Condition to evaluate the attribute against
	// Exactly one condition type must be provided
	// +kubebuilder:validation:Required
	Condition AttributeCondition `json:"condition"`

	// Fallback sampling ratio when the condition doesn't explicitly match
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

// AttributeCondition explicitly supports different attribute types.
type AttributeCondition struct {
	StringCondition  *StringAttributeCondition  `json:"string_condition,omitempty"`
	NumberCondition  *NumberAttributeCondition  `json:"number_condition,omitempty"`
	BooleanCondition *BooleanAttributeCondition `json:"boolean_condition,omitempty"`
	JsonCondition    *JsonAttributeCondition    `json:"json_condition,omitempty"`
}

// StringAttributeCondition for standard string attributes.
type StringAttributeCondition struct {
	// +kubebuilder:validation:Enum=exists;equals;not_equals;contains;not_contains;regex
	Operation string `json:"operation"`

	// Required for all except 'exists'
	ExpectedValue string `json:"expected_value,omitempty"`
}

// JsonAttributeCondition supports operations on JSON serialized as strings
type JsonAttributeCondition struct {
	// +kubebuilder:validation:Enum=exists;is_valid_json;is_invalid_json;equals;not_equals;contains_key;not_contains_key;jsonpath_exists
	Operation string `json:"operation"`

	// ExpectedValue required for equals, not_equals, contains_key, not_contains_key
	ExpectedValue string `json:"expected_value,omitempty"`

	// JsonPath required for jsonpath_exists operation
	JsonPath string `json:"json_path,omitempty"`
}

// NumberAttributeCondition supports numeric types: int, long, float, double
type NumberAttributeCondition struct {
	// +kubebuilder:validation:Enum=exists;equals;not_equals;greater_than;less_than;greater_than_or_equal;less_than_or_equal
	Operation string `json:"operation"`

	// Required for all operations except 'exists'
	ExpectedValue float64 `json:"expected_value,omitempty"`
}

// BooleanAttributeCondition for boolean attribute evaluation
type BooleanAttributeCondition struct {
	// +kubebuilder:validation:Enum=exists;equals
	Operation string `json:"operation"`

	// Required only for 'equals' operation
	ExpectedValue bool `json:"expected_value,omitempty"`
}

type SpanAttributeSamplerStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=spanattributesamplers,scope=Namespaced,shortName=sas
//+kubebuilder:metadata:labels=odigos.io/config=1
//+kubebuilder:metadata:labels=odigos.io/system-object=true

type SpanAttributeSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpanAttributeSamplerSpec   `json:"spec,omitempty"`
	Status SpanAttributeSamplerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type SpanAttributeSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpanAttributeSampler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpanAttributeSampler{}, &SpanAttributeSamplerList{})
}
