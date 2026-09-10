package instrumentationrules

// +kubebuilder:object:generate=true
type CustomProbeReport struct {
	// RuntimeID identifies one native instrumentation lifetime, not a process ID.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	RuntimeID string `json:"runtimeId"`
	// Revision is monotonic only within RuntimeID.
	// +kubebuilder:validation:Minimum=0
	Revision uint64 `json:"revision"`
	// Missing generations are unknown. They never acknowledge retirement.
	Truncated bool `json:"truncated,omitempty"`
	// +kubebuilder:validation:MaxItems=128
	// +listType=map
	// +listMapKey=probe
	// +listMapKey=generation
	Probes []CustomProbeStatus `json:"probes"`
}

// +kubebuilder:object:generate=true
type CustomProbeStatus struct {
	// Revision identifies this transition within the native runtime lifetime.
	// +kubebuilder:validation:Minimum=1
	Revision uint64 `json:"revision"`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	Probe string `json:"probe"`
	// +kubebuilder:validation:Pattern=`^[0-9a-f]{32}$`
	Generation string `json:"generation"`
	// +kubebuilder:validation:Format=date-time
	// +kubebuilder:validation:MaxLength=35
	ExpiresAt string `json:"expiresAt,omitempty"`
	// Installed and retired describe native resources, not telemetry delivery.
	// +kubebuilder:validation:Enum=pending;installed;draining;retired;failed
	State string `json:"state"`
	// +kubebuilder:validation:MaxLength=64
	Reason string `json:"reason"`
	// +kubebuilder:validation:MaxLength=1024
	Message string `json:"message,omitempty"`
	// +kubebuilder:validation:Format=date-time
	// +kubebuilder:validation:MaxLength=35
	UpdatedAt string `json:"updatedAt"`
}
