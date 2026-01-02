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
		pipeline := Pipeline{Exporters: []string{exporterName}}

		if val, ok := dest.GetConfig()[hyperdxLogNormalizer]; ok && getBooleanConfig(val, "true") {
			processorName := "transform/hyperdx-log-normalizer-" + dest.GetID()
			cfg.Processors[processorName] = HyperdxLogNormalizerProcessor
			pipeline.Processors = []string{processorName}
		}

		cfg.Service.Pipelines[pipeName] = pipeline
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
