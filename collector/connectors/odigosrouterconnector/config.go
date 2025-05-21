package odigosrouterconnector

import (
	"github.com/odigos-io/odigos/common/pipelinegen"
	"go.opentelemetry.io/collector/component"
)

type Config struct {
	component.Config
	Groups []pipelinegen.GroupDetails `mapstructure:"groups"`
}

func (c *Config) Validate() error {
	return nil
}
