package odigospiimaskingprocessor

import (
	"go.opentelemetry.io/collector/component"

	"github.com/odigos-io/odigos/common/api/actions"
)

type Config struct {
	actions.PiiMaskingConfig `mapstructure:",squash"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	return nil
}
