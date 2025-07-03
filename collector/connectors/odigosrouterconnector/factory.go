package odigosrouterconnector

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
)

var typeStr = component.MustNewType("odigosrouterconnector")

func NewFactory() connector.Factory {
	return connector.NewFactory(
		component.Type(typeStr),
		createDefaultConfig,
		connector.WithTracesToTraces(createTracesConnector, component.StabilityLevelAlpha),
		connector.WithMetricsToMetrics(createMetricsConnector, component.StabilityLevelAlpha),
		connector.WithLogsToLogs(createLogsConnector, component.StabilityLevelAlpha),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}
