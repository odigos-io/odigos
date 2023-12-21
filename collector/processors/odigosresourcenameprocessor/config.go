package odigosresourcenameprocessor

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig"
	"go.opentelemetry.io/collector/component"
)

// Config defines configuration for Resource processor.
type Config struct {
	k8sconfig.APIConfig `mapstructure:",squash"`
}

var _ component.Config = (*Config)(nil)
