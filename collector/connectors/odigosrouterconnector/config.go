package odigosrouterconnector

import (
	"go.opentelemetry.io/collector/component"

	"github.com/odigos-io/odigos/common/pipelinegen"
)

type Config struct {
	component.Config
	Groups []pipelinegen.GroupDetails `mapstructure:"groups"`
}

func (c *Config) Validate() error {
	return nil
}
