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
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// InstrumentationRuleVerified indicates that the InstrumentationRule has been successfully verified.
	InstrumentationRuleVerified = "InstrumentationRuleValidation"
)

// Includes the instrumentation library name, span kind (for golang) and language
// which identifies a specific library globally.
type InstrumentationLibraryGlobalId struct {

	// The name of the instrumentation library
	Name string `json:"name"`

	// SpanKind is only supported by Golang and will be ignored for any other SDK language.
	// In Go, SpanKind is used because the same instrumentation library can be utilized for different span kinds (e.g., client/server).
	SpanKind common.SpanKind `json:"spanKind,omitempty"`

	// The language in which this library will collect data
	Language common.ProgrammingLanguage `json:"language"`
}

type InstrumentationRuleSpec struct {

	// Allows you to attach a meaningful name to the rule for convenience. Odigos does not use or assume any meaning from this field.
	RuleName string `json:"ruleName,omitempty"`

	// A free-form text field that allows you to attach notes regarding the rule for convenience. For example: why it was added. Odigos does not use or assume any meaning from this field.
	Notes string `json:"notes,omitempty"`

	// A boolean field allowing to temporarily disable the rule, but keep it around for future use
	Disabled bool `json:"disabled,omitempty"`

	// An array of workload objects (name, namespace, kind) to which the rule should be applied. If not specified, the rule will be applied to all workloads. empty array will render the rule inactive.
	Workloads *[]k8sconsts.PodWorkload `json:"workloads,omitempty"`

	// For fine grained control, the user can specify the instrumentation library to use.
	// One can specify same rule for multiple languages and libraries at the same time.
	// If nil, all instrumentation libraries will be used.
	// If empty, no instrumentation libraries will be used.
	InstrumentationLibraries *[]InstrumentationLibraryGlobalId `json:"instrumentationLibraries,omitempty"`

	// Allows to configure payload collection aspects for different types of payloads.
	PayloadCollection *instrumentationrules.PayloadCollection `json:"payloadCollection,omitempty"`

	// Deprecated: use OtelDistros instead.
	OtelSdks *instrumentationrules.OtelSdks `json:"otelSdks,omitempty"`

	// Set the otel distros to use instead of the defaults.
	OtelDistros *instrumentationrules.OtelDistros `json:"otelDistros,omitempty"`

	// Configure which code attributes should be recorded as span attributes.
	CodeAttributes *instrumentationrules.CodeAttributes `json:"codeAttributes,omitempty"`

	// Allows to configure the collection of http headers for different types of payloads.
	HeadersCollection *instrumentationrules.HttpHeadersCollection `json:"headersCollection,omitempty"`

	// Configure the tracing configuration for the library.
	TraceConfig *instrumentationrules.TraceConfig `json:"traceConfig,omitempty"`

	// Add custom instrumentation probes
	CustomInstrumentations *instrumentationrules.CustomInstrumentations `json:"customInstrumentations,omitempty"`

	// Configure general fraction of traces to record if none of the rules evaluate to true.
	// this can help in reducing very noisy and heavy traffic services.
	// note that traces will be dropped regardless of thier attributes/errors/importance.
	HeadSamplingFallbackFraction *instrumentationrules.HeadSamplingFallbackFraction `json:"headSamplingFallbackFraction,omitempty"`
}

// Verify validates the InstrumentationRuleSpec.
// for future usage, you can add more validation logic here for other fields.
func (irs *InstrumentationRuleSpec) Verify() error {
	if irs.CustomInstrumentations != nil {
		return irs.CustomInstrumentations.Verify()
	}
	return nil
}

type InstrumentationRuleStatus struct {
	// Represents the observations of a instrumentationrule's current state.
	// Known .status.conditions.type are: "Available", "Progressing"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:metadata:labels=odigos.io/system-object=true

type InstrumentationRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentationRuleSpec   `json:"spec,omitempty"`
	Status InstrumentationRuleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type InstrumentationRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentationRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstrumentationRule{}, &InstrumentationRuleList{})
}
