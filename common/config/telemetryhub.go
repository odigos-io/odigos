package config

import (
	"github.com/odigos-io/odigos/common"
)

type TelemetryHub struct{}

func (j *TelemetryHub) DestType() common.DestinationType {
	return common.TelemetryHubDestinationType
}

func (j *TelemetryHub) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	uniqueUri := "telemetryhub-" + dest.GetID()
	var pipelineNames []string

	exporterName := "otlp/" + uniqueUri
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint": "https://otlp.telemetryhub.com:4317",
		"headers": GenericMap{
			"x-telemetryhub-key": "${TELEMETRY_HUB_API_KEY}",
		},
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
