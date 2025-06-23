package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CustomInstrumentations struct {
	// Custom instrumentation probes to be added to the SDK.
	Probes []Probe `json:"probes,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type Probe struct {
	ClassName  string `json:"className,omitempty"`
	MethodName string `json:"methodName,omitempty"`
}
