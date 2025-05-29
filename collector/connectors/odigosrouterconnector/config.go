package odigosrouterconnector

import (
	"go.opentelemetry.io/collector/component"

	"github.com/odigos-io/odigos/common/pipelinegen"
)

type Config struct {
	component.Config
	DataStreams []pipelinegen.DataStreams `mapstructure:"dataStreams"`
}

func (c *Config) Validate() error {
	return nil
}
