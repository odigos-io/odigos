package odigosrek8ssourcesexention

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/odigos-io/odigos/collector/extension/odigosrek8ssourcesexention/internal/metadata"
)

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
	return NewOdigosKsResources(set.TelemetrySettings)
}
