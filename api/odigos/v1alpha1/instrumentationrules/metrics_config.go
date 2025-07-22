package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type MetricsConfig struct {
	// Enabled will enable metrics for the rule.
	Enabled bool `json:"disabled,omitempty"`
}
