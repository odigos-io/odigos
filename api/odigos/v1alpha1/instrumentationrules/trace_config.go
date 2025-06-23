package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type TraceConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}
