package config

import (
	"github.com/odigos-io/odigos/common"
)

type LaunchDarkly struct{}

func (j *LaunchDarkly) DestType() common.DestinationType {
	return common.LaunchDarklyDestinationType
}

func (j *LaunchDarkly) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	var pipelineNames []string
	uniqueUri := "launchdarkly-" + dest.GetID()

	exporterName := "otlphttp/" + uniqueUri
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint":    "https://otel.observability.app.launchdarkly.com",
		"compression": "gzip",
		"sending_queue": GenericMap{
			"batch": GenericMap{
				"sizer":    "items",
				"min_size": 512,
				"max_size": 512,
			},
		},
	}

	processorName := "resource/" + uniqueUri
	currentConfig.Processors[processorName] = GenericMap{
		"attributes": []GenericMap{
			{
				"key":    "launchdarkly.project_id",
				"value":  SecretEnvPlaceholder("LAUNCHDARKLY_SDK_KEY", dest),
				"action": "upsert",
			},
		},
	}

	if isTracingEnabled(dest) {
		pipeName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isMetricsEnabled(dest) {
		pipeName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	if isLoggingEnabled(dest) {
		pipeName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[pipeName] = Pipeline{
			Exporters:  []string{exporterName},
			Processors: []string{processorName},
		}
		pipelineNames = append(pipelineNames, pipeName)
	}

	return pipelineNames, nil
}
