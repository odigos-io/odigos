package config

import (
	"github.com/odigos-io/odigos/common"
)

type Debug struct{}

func (s *Debug) DestType() common.DestinationType {
	return common.DebugDestinationType
}

func (s *Debug) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	exporterName := "debug"

	currentConfig.Exporters[exporterName] = GenericMap{
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/debug-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/debug-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/debug-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
