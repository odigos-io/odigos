package config

import (
	"github.com/odigos-io/odigos/common"
)

type Nop struct{}

func (s *Nop) DestType() common.DestinationType {
	return common.NopDestinationType
}

func (s *Nop) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	exporterName := "nop/" + dest.GetID()

	currentConfig.Exporters[exporterName] = GenericMap{}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/nop-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/nop-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/nop-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
