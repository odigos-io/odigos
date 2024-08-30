package v1alpha1

import (
	"github.com/odigos-io/odigos/common"
	"go.opentelemetry.io/otel/attribute"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InstrumentationConfig is the Schema for the instrumentationconfig API
type InstrumentationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec InstrumentationConfigSpec `json:"spec,omitempty"`
}

// Config for the OpenTelemeetry SDKs that should be applied to a workload.
// The workload is identified by the owner reference
type InstrumentationConfigSpec struct {
	// true when the runtime details are invalidated and should be recalculated
	RuntimeDetailsInvalidated bool `json:"runtimeDetailsInvalidated,omitempty"`

	// Configuration for the OpenTelemetry SDKs that this workload should use.
	// The SDKs are identified by the programming language they are written in.
	// TODO: consider adding more granular control over the SDKs, such as community/enterprise, native/ebpf.
	SdkConfigs []SdkConfig `json:"sdkConfigs,omitempty"`
}

type SdkConfig struct {

	// The language of the SDK being configured
	Language common.ProgrammingLanguage `json:"language"`

	// configurations for the instrumentation libraries the the SDK should use
	InstrumentationLibraryConfigs []InstrumentationLibraryConfig `json:"instrumentationLibraryConfigs"`

	// HeadSamplingConfig is a set sampling rules.
	// This config currently only applies to root spans.
	// In the Future we might add another level of configuration base on the parent span (ParentBased Sampling)
	HeadSamplingConfig HeadSamplingConfig `json:"headSamplerConfig,omitempty"`
}

// 'Operand' represents the attributes and values that an operator acts upon in an expression
type AttributeCondition struct {
	// attribute key (e.g. "url.path")
	Key attribute.Key `json:"key"`
	// currently only string values are supported.
	Val string `json:"val"`
	// The operator to use to compare the attribute value.
	Operator Operator `json:"operator,omitempty"`
}

// +kubebuilder:validation:Enum=equals;notEquals;endWith;startWith
// +kubebuilder:default:=equals
type Operator string

const (
	Equals    Operator = "equals"
	NotEquals Operator = "notEquals"
	EndWith   Operator = "endWith"
	StartWith Operator = "startWith"
)

// AttributesAndSamplerRule is a set of AttributeCondition that are ANDed together.
// If all attribute conditions evaluate to true, the AND sampler evaluates to true,
// and the fraction is used to determine the sampling decision.
// If any of the attribute compare samplers evaluate to false,
// the fraction is not used and the rule is skipped.
// An "empty" AttributesAndSamplerRule with no attribute conditions is considered to always evaluate to true.
// and the fraction is used to determine the sampling decision.
// This entity is refered to a rule in Odigos terminology for head-sampling.
type AttributesAndSamplerRule struct {
	AttributeConditions []AttributeCondition `json:"attributeConditions"`
	// The fraction of spans to sample, in the range [0, 1].
	// If the fraction is 0, no spans are sampled.
	// If the fraction is 1, all spans are sampled.
	// +kubebuilder:default:=1
	Fraction float64 `json:"fraction"`
}

// HeadSamplingConfig is a set of attribute rules.
// The first attribute rule that evaluates to true is used to determine the sampling decision based on its fraction.
//
// If none of the rules evaluate to true, the fallback fraction is used to determine the sampling decision.
type HeadSamplingConfig struct {
	AttributesAndSamplerRules []AttributesAndSamplerRule `json:"attributesAndSamplerRules"`
	// Used as a fallback if all rules evaluate to false,
	// it may be empty - in this case the default value will be 1 - all spans are sampled.
	// it should be a float value in the range [0, 1] - the fraction of spans to sample.
	// a value of 0 means no spans are sampled if none of the rules evaluate to true.
	// +kubebuilder:default:=1
	FallbackFraction float64 `json:"fallbackFraction"`
}

// The InstrumentationLibraryCapabilityParameter represents a single configuration property that can be set for an instrumentation library option.
// For example, the capability might be payload collection, and the parameter can be the maximum size of the payload to record,
// if to skip recording incomplete payloads, conditions for recording payload (for example only for specific mime types - "only application/json"),
// additional attributes to record original length, etc.
type InstrumentationLibraryCapabilityParameter struct {
	// The name of the property that should be configured for the instrumentation library option
	ParameterName string `json:"parameterName"`

	// If this parameter value is boolean, the boolean value should be set here.
	// Default value for any boolean parameter is false, which can be omitted in this case.
	BooleanValue *bool `json:"booleanValue,omitempty"`

	// When the value is an integer, the integer value should be set here.
	// For example: number of bytes, number of items, etc.
	// If absent, the instrumentation library can use a pre-defined default value
	// which can be any number that makes sense for the parameter (not necessarily 0).
	// To use a value 0 and not the default, set the value to 0 instead of omitting it.
	IntValue *int `json:"intValue,omitempty"`

	// If this parameter value is for a number, the number value should be set here.
	// If absent, the instrumentation library can use a pre-defined default value
	// which can be any number that makes sense for the parameter (not necessarily 0).
	// To use a value 0 and not the default, set the value to 0 instead of omitting it.
	NumberValue *float64 `json:"numberValue,omitempty"`

	// StringValue is used for string parameters.
	// If absent, the instrumentation library can use a pre-defined default value (not necessarily empty string).
	// To use an empty string, set the value to an empty string instead of omitting it.
	StringValue *string `json:"stringValue,omitempty"`

	// If the parameter value is a list of strings, the list should be set here.
	// If absent, the instrumentation library can use a pre-defined default value (not necessarily an empty list).
	// To use an empty list, set the value to an empty list instead of omitting it.
	StringListValue []string `json:"stringListValue,omitempty"`
}

// Each instrumentation library can implement a set of capabilities that can be configured by the user.
// The capabilities should be published by the instrumentation library and documented elsewhere.
// If the capability is not used, it should be omitted from the configuration.
type InstrumentationLibraryCapability struct {

	// Each instrumentation library advertise a set of capabilities that can be configured be the users.
	// The capability name is used to identify the capability that is being configured.
	CapabilityName string `json:"capabilityName"`

	// The parameters that can be configured for the capability, which are specific to the capability and the instrumentation library.
	// Used to configure the behavior of the capability.
	Parameters []InstrumentationLibraryCapabilityParameter `json:"parameters,omitempty"`
}

type InstrumentationLibraryConfig struct {
	InstrumentationLibraryId InstrumentationLibraryId `json:"libraryId"`

	TraceConfig *InstrumentationLibraryConfigTraces `json:"traceConfig,omitempty"`

	// A list of enabled capabilities for the instrumentation library and their configuration.
	Capabilities []InstrumentationLibraryCapability `json:"capabilities,omitempty"`
}

type InstrumentationLibraryId struct {
	// The name of the instrumentation library
	// - Node.js: The name of the npm package: `@opentelemetry/instrumentation-<name>`
	InstrumentationLibraryName string `json:"libraryName"`
	// SpanKind is only supported by Golang and will be ignored for any other SDK language.
	// In Go, SpanKind is used because the same instrumentation library can be utilized for different span kinds (e.g., client/server).
	SpanKind common.SpanKind `json:"spanKind,omitempty"`
}

type InstrumentationLibraryConfigTraces struct {
	// Whether the instrumentation library is enabled to record traces.
	// When false, it is expected that the instrumentation library does not produce any spans regardless of any other configuration.
	// When true, the instrumentation library should produce spans according to the other configuration options.
	// If not specified, the default value for this signal should be used (whether to enable libraries by default or not).
	Enabled *bool `json:"enabled,omitempty"`
}

// +kubebuilder:object:root=true

// InstrumentationConfigList contains a list of InstrumentationOption
type InstrumentationConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentationConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstrumentationConfig{}, &InstrumentationConfigList{})
}
