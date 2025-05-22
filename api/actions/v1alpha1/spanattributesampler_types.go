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
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpanAttributeSamplerSpec defines the desired state of SpanAttributeSampler
type SpanAttributeSamplerSpec struct {
	// ActionName is the name of the sampling action. This may be used to
	// describe the purpose or intent of this sampler, for documentation
	// or reference within other tools or systems.
	ActionName string `json:"actionName,omitempty"`

	// Notes provides additional, free-form information about this sampler,
	// such as a reference to a ticket, a link, or usage guidelines.
	Notes string `json:"notes,omitempty"`

	// Disabled, if set to true, indicates that this sampler should not be
	// applied at runtime.
	Disabled bool `json:"disabled,omitempty"`

	// Signals indicates which ObservabilitySignal types this sampler applies to.
	// For instance, this could include traces, metrics, logs, etc.
	Signals []common.ObservabilitySignal `json:"signals"`

	// AttributeFilters defines a list of criteria to decide how spans should be
	// sampled based on their attributes. At least one filter is required.
	// +kubebuilder:validation:Required
	AttributeFilters []SpanAttributeFilter `json:"attribute_filters"`
}

// SpanAttributeFilter allows sampling traces based on specific span attributes and conditions.
type SpanAttributeFilter struct {
	// ServiceName specifies which service this filter applies to. Only spans
	// originating from the given service will be evaluated against this filter.
	// +kubebuilder:validation:Required
	ServiceName string `json:"service_name"`

	// AttributeKey indicates which attribute on the span to evaluate.
	// +kubebuilder:validation:Required
	AttributeKey string `json:"attribute_key"`

	// Condition is the rule or expression that will be used to evaluate
	// the attribute's value. Exactly one of the condition types must be set:
	//   - StringCondition
	//   - NumberCondition
	//   - BooleanCondition
	//   - JsonCondition
	// +kubebuilder:validation:Required
	Condition AttributeCondition `json:"condition"`

	// SamplingRatio determines the percentage (0–100) of traces to sample
	// when the specified attribute satisfies the filter.
	//
	// For example, a value of 100 means all such traces will be kept,
	// while a value of 0 means all will be dropped.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=100
	SamplingRatio float64 `json:"sampling_ratio"`

	// FallbackSamplingRatio is the percentage (0–100) of spans to sample
	// when the condition does not explicitly match. For example, if set to 50,
	// then half of non-matching spans would be sampled.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=100
	FallbackSamplingRatio float64 `json:"fallback_sampling_ratio"`
}

// AttributeCondition wraps different condition types so that only one type
// of condition needs to be specified. This makes it explicit which data type
// the attribute is expected to have.
type AttributeCondition struct {
	// StringCondition applies to string-type attributes.
	StringCondition *StringAttributeCondition `json:"string_condition,omitempty"`

	// NumberCondition applies to numeric attributes (int, long, float, double).
	NumberCondition *NumberAttributeCondition `json:"number_condition,omitempty"`

	// BooleanCondition applies to boolean attributes.
	BooleanCondition *BooleanAttributeCondition `json:"boolean_condition,omitempty"`

	// JsonCondition applies to attributes that are JSON-encoded strings.
	JsonCondition *JsonAttributeCondition `json:"json_condition,omitempty"`
}

// StringAttributeCondition defines how to evaluate a string attribute.
type StringAttributeCondition struct {
	// Operation determines what comparison or check should be performed.
	//
	// The valid operations are:
	//
	//   - "exists": Checks that the attribute is present (and not an empty string).
	//   - "equals": String equality comparison with ExpectedValue.
	//   - "not_equals": String inequality comparison with ExpectedValue.
	//   - "contains": Checks if the attribute contains ExpectedValue as a substring.
	//   - "not_contains": Checks if the attribute does not contain ExpectedValue.
	//   - "regex": Interprets ExpectedValue as a regular expression (RE2 syntax)
	//       and checks for a match within the attribute.
	//
	// For operations other than "exists", ExpectedValue must be provided.
	// +kubebuilder:validation:Enum=exists;equals;not_equals;contains;not_contains;regex
	Operation string `json:"operation"`

	// ExpectedValue is required for all operations except "exists". Its usage
	// depends on the chosen Operation, e.g. it may represent an exact string
	// to match, a substring, or a regular expression.
	ExpectedValue string `json:"expected_value,omitempty"`
}

// JsonAttributeCondition defines conditions for evaluating JSON-encoded span attributes.
// It supports structural and value-based filtering using key lookups and JSONPath expressions.
//
// Supported operations:
//   - "exists": Checks that the attribute is a non-empty string.
//   - "is_valid_json": Ensures the string parses as valid JSON.
//   - "is_invalid_json": Ensures the string is not valid JSON.
//   - "equals": Compares the full JSON string to expected_value.
//   - "not_equals": Verifies the full JSON string is not equal to expected_value.
//   - "jsonpath_exists": Asserts that the specified json_path resolves to a non-empty result.
//   - "key_equals": Compares the value at json_path to expected_value.
//   - "key_not_equals": Verifies the value at json_path is not equal to expected_value.
type JsonAttributeCondition struct {
	// Operation defines the evaluation logic applied to the JSON value.
	//
	// Supported values:
	//   - is_valid_json
	//   - is_invalid_json
	//   - equals
	//   - not_equals
	//   - jsonpath_exists
	//   - key_equals
	//   - key_not_equals
	//
	// +kubebuilder:validation:Enum=is_valid_json;is_invalid_json;equals;not_equals;jsonpath_exists;key_equals;key_not_equals
	Operation string `json:"operation"`

	// JsonPath is required for:
	//   - jsonpath_exists
	//   - key_equals
	//   - key_not_equals
	//
	// It is a JSONPath expression used to navigate the JSON structure.
	// Example: $.user.role or $.events[0].code
	JsonPath string `json:"json_path,omitempty"`

	// ExpectedValue is required for:
	//   - equals
	//   - not_equals
	//   - key_equals
	//   - key_not_equals
	//
	// The value to compare against during evaluation.
	ExpectedValue string `json:"expected_value,omitempty"`
}

// NumberAttributeCondition applies to attributes that are numeric (int, float, etc.).
type NumberAttributeCondition struct {
	// Operation determines the numeric comparison to perform.
	//
	// Valid operations:
	//
	//   - "exists": Checks that the numeric attribute is present (non-null).
	//   - "equals": Checks if the attribute equals ExpectedValue.
	//   - "not_equals": Checks if the attribute does not equal ExpectedValue.
	//   - "greater_than": Checks if attribute > ExpectedValue.
	//   - "less_than": Checks if attribute < ExpectedValue.
	//   - "greater_than_or_equal": Checks if attribute >= ExpectedValue.
	//   - "less_than_or_equal": Checks if attribute <= ExpectedValue.
	//
	// For operations other than "exists", ExpectedValue must be specified.
	// +kubebuilder:validation:Enum=exists;equals;not_equals;greater_than;less_than;greater_than_or_equal;less_than_or_equal
	Operation string `json:"operation"`

	// ExpectedValue is required for all operations except "exists".
	ExpectedValue float64 `json:"expected_value,omitempty"`
}

// BooleanAttributeCondition defines a check against a boolean attribute.
type BooleanAttributeCondition struct {
	// Operation can be:
	//   - "exists": Checks that the boolean attribute is present.
	//   - "equals": Checks if the attribute exactly matches ExpectedValue.
	//
	// ExpectedValue is required only for the "equals" operation.
	// +kubebuilder:validation:Enum=exists;equals
	Operation string `json:"operation"`

	// ExpectedValue is only used if Operation == "equals".
	ExpectedValue bool `json:"expected_value,omitempty"`
}

// SpanAttributeSamplerStatus represents the current status of a SpanAttributeSampler.
type SpanAttributeSamplerStatus struct {
	// Conditions is a list of the latest available observations of this sampler's state.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=spanattributesamplers,scope=Namespaced,shortName=sas
// +kubebuilder:metadata:labels=odigos.io/system-object=true
// +kubebuilder:storageversion
// SpanAttributeSampler is the Schema for the spanattributesamplers API.
// It holds the specification for sampling spans based on attribute conditions,
// as well as the sampler's current status.
type SpanAttributeSampler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpanAttributeSamplerSpec   `json:"spec,omitempty"`
	Status SpanAttributeSamplerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// SpanAttributeSamplerList contains a list of SpanAttributeSampler objects.
type SpanAttributeSamplerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpanAttributeSampler `json:"items"`
}
