package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CodeAttributes struct {

	// Should record the `code.column` attribute.
	Column *bool `json:"column,omitempty"`

	// Should record the `code.filepath` attribute.
	FilePath *bool `json:"filePath,omitempty"`

	// Should record the `code.function` attribute.
	Function *bool `json:"function,omitempty"`

	// Should record the `code.lineno` attribute.
	LineNumber *bool `json:"lineNumber,omitempty"`

	// Should record the `code.namespace` attribute.
	Namespace *bool `json:"namespace,omitempty"`

	// Should record the `code.stacktrace` attribute.
	Stacktrace *bool `json:"stackTrace,omitempty"`
}
