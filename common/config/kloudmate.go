package config

import (
	"github.com/odigos-io/odigos/common"
)

type KloudMate struct{}

func (j *KloudMate) DestType() common.DestinationType {
	return common.KloudMateDestinationType
}

func (j *KloudMate) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	uniqueUri := "kloudmate-" + dest.GetID()

	exporterName := "otlphttp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": "https://otel.kloudmate.com:4318",
		"headers": GenericMap{
			"Authorization": "${KLOUDMATE_API_KEY}",
		},
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
