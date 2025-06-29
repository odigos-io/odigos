package config

import (
	"github.com/odigos-io/odigos/common"
)

type GoogleCloud struct{}

func (g *GoogleCloud) DestType() common.DestinationType {
	return common.GoogleCloudDestinationType
}

func (g *GoogleCloud) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	var pipelineNames []string
	if isTracingEnabled(dest) {
		exporterName := "googlecloud/" + dest.GetID()
		currentConfig.Exporters[exporterName] = struct{}{}

		tracesPipelineName := "traces/googlecloud-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isLoggingEnabled(dest) {
		exporterName := "googlecloud/" + dest.GetID()
		currentConfig.Exporters[exporterName] = GenericMap{
			"log": GenericMap{
				"default_log_name": "opentelemetry.io/collector-exported-log",
			},
		}

		logsPipelineName := "logs/googlecloud-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
