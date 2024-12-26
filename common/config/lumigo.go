package config

import (
	"github.com/odigos-io/odigos/common"
)

type Lumigo struct{}

func (j *Lumigo) DestType() common.DestinationType {
	return common.LumigoDestinationType
}

func (j *Lumigo) ModifyConfig(dest ExporterConfigurer, cfg *Config) error {
	uniqueUri := "lumigo-" + dest.GetID()

	exporterName := "otlphttp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": "https://ga-otlp.lumigo-tracer-edge.golumigo.com",
		"headers": GenericMap{
			"Authorization": "LumigoToken ${LUMIGO_TOKEN}",
		},
	}

	cfg.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
