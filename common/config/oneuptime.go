package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	ONEUPTIME_INGESTION_KEY = "ONEUPTIME_INGESTION_KEY"
)

type OneUptime struct{}

func (j *OneUptime) DestType() common.DestinationType {
	return common.OneUptimeDestinationType
}

func (j *OneUptime) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	uniqueUri := "oneuptime-" + dest.GetID()

	exporterName := "otlphttp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": "https://oneuptime.com/otlp",
		"encoding": "json",
		"headers": GenericMap{
			"Content-Type":      "application/json",
			"x-oneuptime-token": "${ONEUPTIME_INGESTION_KEY}",
		},
	}

	currentConfig.Exporters[exporterName] = exporterConfig
	var pipelineNames []string
	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
