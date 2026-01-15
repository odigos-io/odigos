package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

const ActionSpanRenamer = "SpanRenamer"

// configuration for replacing parts of the span name with a template text based on regular expressions.
type SpanRenamerRegexReplacement struct {

	// the text to be used for replacing the matched part of the span name.
	TemplateText string `json:"templateText"`

	// regualr expression that will be used to match the part of the span name to be replaced.
	RegexPattern string `json:"regexPattern"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type SpanRenamerConfig struct {

	// the programming language which the renamed spans are written in.
	ProgrammingLanguage common.ProgrammingLanguage `json:"programmingLanguage"`

	// the name of the opentelemetry intrumentation scope which is producing the spans to be renamed.
	ScopeName string `json:"scopeName"`

	// list of regex replacements to be applied to the span name.
	// all options are always tried, regardless of whether the previous options have matched or not.
	RegexReplacements []SpanRenamerRegexReplacement `json:"regexReplacements,omitempty"`
}

func (SpanRenamerConfig) ProcessorType() string {
	return "odigosspanrenamer"
}

func (SpanRenamerConfig) OrderHint() int {
	return 1
}

func (SpanRenamerConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
