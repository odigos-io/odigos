package instrumentationrules

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type MetricsConfig struct {
	// NetworkMetrics enables network flow metrics for scoped workloads.
	NetworkMetrics *MetricSignal `json:"networkMetrics,omitempty" yaml:"networkMetrics,omitempty"`
	// StatsMetrics enables TCP stats metrics for scoped workloads.
	StatsMetrics *MetricSignal `json:"statsMetrics,omitempty" yaml:"statsMetrics,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type MetricSignal struct {
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

func MetricSignalEnabled(s *MetricSignal) bool {
	return s != nil && s.Enabled != nil && *s.Enabled
}

func (c *MetricsConfig) NetworkMetricsEnabled() bool {
	return MetricSignalEnabled(c.NetworkMetrics)
}

func (c *MetricsConfig) StatsMetricsEnabled() bool {
	return MetricSignalEnabled(c.StatsMetrics)
}

func (c *MetricsConfig) AnyEnabled() bool {
	return c != nil && (c.NetworkMetricsEnabled() || c.StatsMetricsEnabled())
}
