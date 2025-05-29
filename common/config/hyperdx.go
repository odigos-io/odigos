package config

import (
	"github.com/odigos-io/odigos/common"
)

type HyperDX struct{}

func (j *HyperDX) DestType() common.DestinationType {
	return common.HyperDxDestinationType
}

func (j *HyperDX) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	uniqueUri := "hdx-" + dest.GetID()

	exporterName := "otlp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": "in-otel.hyperdx.io:4317",
		"headers": GenericMap{
			"authorization": "${HYPERDX_API_KEY}",
		},
	}

	cfg.Exporters[exporterName] = exporterConfig
	var pipelineNames []string
	if IsTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if IsMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if IsLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		cfg.Service.Pipelines[pipeName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
