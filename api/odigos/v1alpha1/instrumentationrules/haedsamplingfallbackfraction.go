package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type HeadSamplingFallbackFraction struct {
	// The fraction of traces to keep if none of the head sampling rules evaluate to true.
	// should be in range [0, 1]
	// 0 means no traces will be recorded (unless they matched a rule to keep)
	// 1 (default) means all traces will be recorded
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=1
	FractionToKeep float64 `json:"fractionToKeep,omitempty"`
}
