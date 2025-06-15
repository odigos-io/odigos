package config

import (
	"github.com/odigos-io/odigos/destinations"
)

type KloudMate struct{}

func (j *KloudMate) DestType() destinations.DestinationType {
	return destinations.KloudMateDestinationType
}

func (j *KloudMate) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	uniqueUri := "kloudmate-" + dest.GetID()

	exporterName := "otlphttp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": "https://otel.kloudmate.com:4318",
		"headers": GenericMap{
			"Authorization": "${KLOUDMATE_API_KEY}",
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
