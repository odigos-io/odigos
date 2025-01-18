package config

import (
	"github.com/odigos-io/odigos/common"
)

type SumoLogic struct{}

func (s *SumoLogic) DestType() common.DestinationType {
	return common.SumoLogicDestinationType
}

func (s *SumoLogic) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	exporterName := "otlphttp/sumologic-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": "${SUMOLOGIC_COLLECTION_URL}",
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/sumologic-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/sumologic-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/sumologic-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
