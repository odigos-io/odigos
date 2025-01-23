package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CodeAttributes struct {

	// Should record the `code.column` attribute.
	// if unset, the value will resolve from other relevant rules, or fallback to false
	Column *bool `json:"column,omitempty"`

	// Should record the `code.filepath` attribute.
	// if unset, the value will resolve from other relevant rules, or fallback to false
	FilePath *bool `json:"filePath,omitempty"`

	// Should record the `code.function` attribute.
	// if unset, the value will resolve from other relevant rules, or fallback to false
	Function *bool `json:"function,omitempty"`

	// Should record the `code.lineno` attribute.
	// if unset, the value will resolve from other relevant rules, or fallback to false
	LineNumber *bool `json:"lineNumber,omitempty"`

	// Should record the `code.namespace` attribute.
	// if unset, the value will resolve from other relevant rules, or fallback to false
	Namespace *bool `json:"namespace,omitempty"`

	// Should record the `code.stacktrace` attribute.
	// if unset, the value will resolve from other relevant rules, or fallback to false
	Stacktrace *bool `json:"stackTrace,omitempty"`
}
