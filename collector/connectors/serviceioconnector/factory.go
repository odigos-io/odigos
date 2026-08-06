//go:generate mdatagen metadata.yaml

package serviceioconnector

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"

	"github.com/odigos-io/odigos/collector/connectors/serviceioconnector/internal/metadata"
)

func NewFactory() connector.Factory {
	return connector.NewFactory(
		metadata.Type,
		createDefaultConfig,
		connector.WithTracesToMetrics(createTracesToMetricsConnector, metadata.TracesToMetricsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createTracesToMetricsConnector(
	_ context.Context,
	params connector.Settings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (connector.Traces, error) {
	typedCfg := cfg.(*Config)
	if err := typedCfg.Validate(); err != nil {
		return nil, err
	}
	return newConnector(params.TelemetrySettings, typedCfg, nextConsumer)
}
