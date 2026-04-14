package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type EbpfLogCapture struct {
	// Enabled switches the logs pipeline to use only the eBPF receiver,
	// replacing the filelog receiver, and enables eBPF-based log capture
	// in odiglet for instrumented processes.
	Enabled *bool `json:"enabled,omitempty"`
}
