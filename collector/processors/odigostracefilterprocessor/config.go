package odigostracefilterprocessor

import (
	"go.opentelemetry.io/collector/component"
)

// Config defines configuration for the odigos_trace_filter processor.
// Designed for extensibility: additional filtering rules can be added here.
type Config struct {
	// DropUnsampled controls whether spans without the W3C sampled bit set
	// in their flags field should be dropped.
	// Uses bitmask logic: a span is considered sampled if (flags & 1) == 1.
	DropUnsampled bool `mapstructure:"drop_unsampled"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	return nil
}
