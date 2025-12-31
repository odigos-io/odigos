package odigosebpfreceiver

import "time"

type Config struct {
	// MetricsConfig contains configuration specific to metrics collection
	MetricsConfig MetricsConfig `mapstructure:"metrics_config"`
}

// MetricsConfig holds configuration for metrics collection
type MetricsConfig struct {
	// Interval defines how often metrics are collected from eBPF maps.
	// Defaults to 30 seconds if not specified.
	Interval time.Duration `mapstructure:"interval"`
}
