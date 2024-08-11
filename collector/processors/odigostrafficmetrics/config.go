package odigostrafficmetrics

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	// ResourceAttributesKeys is a list of resource attributes keys that will be used to add labels for the metrics.
	ResourceAttributesKeys []string `mapstructure:"res_attributes_keys"`

	// SamplingRatio is the ratio of payloads that are measured. Values between 0.0 and 1.0 are valid.
	// default is 1.0.
	// It is useful to set this value when the processor is used in a high throughput environment
	// and the overhead of measuring the metrics for each span/metric/log is too high.
	SamplingRatio float64 `mapstructure:"sampling_ratio"`
}

var _ component.ConfigValidator = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (c Config) Validate() error {
	if c.SamplingRatio < 0 || c.SamplingRatio > 1 {
		return errors.New("sampling_ratio must be between 0.0 and 1.0")
	}

	return nil
}
