package config

import (
	"github.com/odigos-io/odigos/common"
)

type Debug struct{}

const (
	VERBOSITY = "VERBOSITY"
)

func (s *Debug) DestType() common.DestinationType {
	return common.DebugDestinationType
}

func (s *Debug) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	exporterName := "debug/" + dest.GetID()

	verbosity, exists := dest.GetConfig()[VERBOSITY]
	if !exists {
		// Default verbosity
		verbosity = "basic"
	}

	currentConfig.Exporters[exporterName] = GenericMap{
		"verbosity": verbosity,
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
