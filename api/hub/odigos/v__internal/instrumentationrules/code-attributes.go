package instrumentationrules

type CodeAttributes struct {
	Column     *bool `json:"column,omitempty"`
	FilePath   *bool `json:"filePath,omitempty"`
	Function   *bool `json:"function,omitempty"`
	LineNumber *bool `json:"lineNumber,omitempty"`
	Namespace  *bool `json:"namespace,omitempty"`
	Stacktrace *bool `json:"stackTrace,omitempty"`
}
