package odigoscapabilitiesextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// Type is the extension's component type.
var Type = component.MustNewType("odigos_capabilities")

const stability = component.StabilityLevelDevelopment

func NewFactory() extension.Factory {
	return extension.NewFactory(
		Type,
		createDefaultConfig,
		create,
		stability,
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func create(_ context.Context, set extension.Settings, _ component.Config) (extension.Extension, error) {
	return &capabilitiesExtension{logger: set.Logger}, nil
}
