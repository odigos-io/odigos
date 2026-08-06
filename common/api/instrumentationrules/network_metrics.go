package instrumentationrules

// NetworkMetricsConfig enables network flow and TCP stats metrics for scoped workloads.
// Enablement is presence-based: a non-nil value means network metrics are collected,
// nil means they are not. Collection settings (attributes, flush interval, etc.) will
// be added as fields here.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type NetworkMetricsConfig struct {
}
