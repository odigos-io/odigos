package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type TraceConfig struct {
	// Disabled will disable tracing for the rule.
	Disabled *bool `json:"enabled,omitempty"`
}
