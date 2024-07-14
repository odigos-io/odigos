package odigostrafficmetrics

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

type Config struct {
	// ResourceAttributesKeys is a list of resource attributes keys that will be used to add labels for the metrics.
	ResourceAttributesKeys []string `mapstructure:"res_attributes_keys"`
}

var _ component.ConfigValidator = (*Config)(nil)

var (
	errEmptyKeys       = errors.New("resource_attributes_keys must not be empty")
)

// Validate checks if the processor configuration is valid
func (c Config) Validate() error {
	if len(c.ResourceAttributesKeys) == 0 {
		return errEmptyKeys
	}

	return nil
}
