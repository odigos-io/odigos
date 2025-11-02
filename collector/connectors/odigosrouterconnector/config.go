package odigosrouterconnector

import (
	"go.opentelemetry.io/collector/component"

	"github.com/odigos-io/odigos/common/pipelinegen"
)

type Config struct {
	component.Config
	DataStreams                   []pipelinegen.DataStreams `mapstructure:"datastreams"`
	OdigosK8sResourcesExtensionID component.ID              `mapstructure:"odigos_k8s_resources_extension_id"`
}

func (c *Config) Validate() error {
	return nil
}
