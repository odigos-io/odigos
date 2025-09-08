package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	gcpProjectIdKey        = "GCP_PROJECT_ID"
	gcpBillingProjectIdKey = "GCP_BILLING_PROJECT"
	gcpTimeoutKey          = "GCP_TIMEOUT"
)

type GoogleCloud struct{}

func (g *GoogleCloud) DestType() common.DestinationType {
	return common.GoogleCloudDestinationType
}

func (g *GoogleCloud) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	var pipelineNames []string
	if isTracingEnabled(dest) {
		exporterName := "googlecloud/" + dest.GetID()
		exporterConfig := GenericMap{}
		if projectId, exists := dest.GetConfig()[gcpProjectIdKey]; exists {
			exporterConfig["project_id"] = projectId
		}
		if billingProjectId, exists := dest.GetConfig()[gcpBillingProjectIdKey]; exists {
			exporterConfig["billing_project_id"] = billingProjectId
		}
		if timeout, exists := dest.GetConfig()[gcpTimeoutKey]; exists {
			exporterConfig["timeout"] = timeout
		}

		currentConfig.Exporters[exporterName] = exporterConfig

		tracesPipelineName := "traces/googlecloud-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isLoggingEnabled(dest) {
		exporterName := "googlecloud/" + dest.GetID()
		exporterConfig := GenericMap{
			"log": GenericMap{
				"default_log_name": "opentelemetry.io/collector-exported-log",
			},
		}
		if timeout, exists := dest.GetConfig()[gcpTimeoutKey]; exists {
			exporterConfig["timeout"] = timeout
		}

		if projectId, exists := dest.GetConfig()[gcpProjectIdKey]; exists {
			exporterConfig["project_id"] = projectId
		}
		if billingProjectId, exists := dest.GetConfig()[gcpBillingProjectIdKey]; exists {
			exporterConfig["billing_project_id"] = billingProjectId
		}

		currentConfig.Exporters[exporterName] = exporterConfig

		logsPipelineName := "logs/googlecloud-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
