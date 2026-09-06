package odigosrouterconnector

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector/xconnector"
)

var typeStr = component.MustNewType("odigosrouterconnector")

func NewFactory() xconnector.Factory {
	return xconnector.NewFactory(
		component.Type(typeStr),
		createDefaultConfig,
		xconnector.WithTracesToTraces(createTracesConnector, component.StabilityLevelAlpha),
		xconnector.WithMetricsToMetrics(createMetricsConnector, component.StabilityLevelAlpha),
		xconnector.WithLogsToLogs(createLogsConnector, component.StabilityLevelAlpha),
		xconnector.WithProfilesToProfiles(createProfilesConnector, component.StabilityLevelAlpha),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}
