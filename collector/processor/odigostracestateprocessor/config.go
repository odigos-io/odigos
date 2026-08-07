package odigostracestateprocessor

import (
	"go.opentelemetry.io/collector/component"

	"github.com/odigos-io/odigos/common/api/sampling"
)

type Config struct {
	SpanSamplingAttributes *sampling.SpanSamplingAttributesConfiguration `mapstructure:"span_sampling_attributes"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	return nil
}

func (cfg *Config) isCategoryEnabled() bool {
	return cfg.SpanSamplingAttributes == nil ||
		cfg.SpanSamplingAttributes.SamplingCategoryDisabled == nil ||
		!*cfg.SpanSamplingAttributes.SamplingCategoryDisabled
}

func (cfg *Config) isTraceDecidingRuleEnabled() bool {
	return cfg.SpanSamplingAttributes == nil ||
		cfg.SpanSamplingAttributes.TraceDecidingRuleDisabled == nil ||
		!*cfg.SpanSamplingAttributes.TraceDecidingRuleDisabled
}
