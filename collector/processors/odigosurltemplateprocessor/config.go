package odigosurltemplateprocessor

import (
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type Config struct {
}

var _ xconfmap.Validator = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (c Config) Validate() error {
	return nil
}
