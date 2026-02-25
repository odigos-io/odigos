package odigosworkloadconfigextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/odigos-io/odigos/collector/extension/odigosworkloadconfigextension/internal/metadata"
)

// Type is the extension's component type. Use with component.NewID(Type) to obtain the extension from the host.
var Type = metadata.Type

//go:generate mdatagen metadata.yaml

func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		create,
		metadata.ExtensionStability,
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func create(_ context.Context, set extension.Settings, _ component.Config) (extension.Extension, error) {
	return NewOdigosConfig(set.TelemetrySettings)
}
