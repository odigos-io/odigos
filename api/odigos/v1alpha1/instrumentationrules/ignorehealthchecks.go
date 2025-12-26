package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type IgnoreHealthChecks struct {
	// How many health checks traces to record
	// should be in range [0, 1]
	// 0 (default) means no health checks traces will be recorded
	// 1 means all health checks traces will be recorded
	// +kubebuilder:default:=0
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=1
	FractionToRecord float64 `json:"fractionToRecord,omitempty"`
}
