package actions

// configuration for replacing parts of the span name with a template text based on regular expressions.
type SpanRenamerRegexReplacement struct {
	// the text to be used for replacing the matched part of the span name.
	TemplateText string `json:"templateText"`

	// regular expression that will be used to match the part of the span name to be replaced.
	RegexPattern string `json:"regexPattern"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type SpanRenamerScopeRules struct {
	// the name of the opentelemetry intrumentation scope which the renamed spans are written in.
	ScopeName string `json:"scopeName"`

	// list of regex replacements to be applied to the span name.
	// all options are always tried, regardless of whether the previous options have matched or not.
	RegexReplacements []SpanRenamerRegexReplacement `json:"regexReplacements,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type SpanRenamerConfig struct {
	// list of scope rules to be applied to the span name.
	// all options are always tried, regardless of whether the previous options have matched or not.
	ScopeRules []SpanRenamerScopeRules `json:"scopeRules,omitempty"`
}

type SpanRenamerScopeConfig struct {
	// the name of the opentelemetry intrumentation scope which the renamed spans are written in.
	ScopeName string `json:"scopeName"`

	// if set, spans matching the above conditions will be renamed to this static value.
	ConstantSpanName string `json:"constantSpanName,omitempty"`
}
