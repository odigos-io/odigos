package config

import (
	"github.com/odigos-io/odigos/common"
)

type GoogleCloudOTLP struct{}

func (g *GoogleCloudOTLP) DestType() common.DestinationType {
	return common.GoogleCloudOTLPDestinationType
}

func (g *GoogleCloudOTLP) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	var pipelineNames []string
	if isTracingEnabled(dest) {
		exporterName := "otlphttp/gcp-" + dest.GetID()
		extensionName := "googleclientauth/" + dest.GetID()
		config := dest.GetConfig()
		exporterConfig := GenericMap{
			"encoding": "proto",
			"endpoint": "https://telemetry.googleapis.com",
			"auth": GenericMap{
				"authenticator": extensionName,
			},
		}
		currentConfig.Exporters[exporterName] = exporterConfig

		tracesPipelineName := "traces/googlecloudotlp-" + dest.GetID()
		pipeline := Pipeline{
			Exporters: []string{exporterName},
		}

		if _, exists := config[gcpProjectIdKey]; exists {
			processorName := "resource/gcp-" + dest.GetID()
			processorConfig := GenericMap{
				"attributes": []GenericMap{
					{
						"key":    "gcp.project_id",
						"value":  config[gcpProjectIdKey],
						"action": "insert",
					},
				},
			}
			currentConfig.Processors[processorName] = processorConfig
			pipeline.Processors = []string{processorName}
		}

		currentConfig.Service.Pipelines[tracesPipelineName] = pipeline

		extensionConfig := GenericMap{}
		if val, exists := config[gcpProjectIdKey]; exists {
			extensionConfig["project"] = val
		}
		if val, exists := config[gcpBillingProjectIdKey]; exists {
			extensionConfig["quota_project"] = val
		}
		currentConfig.Extensions[extensionName] = extensionConfig
		currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, extensionName)
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
