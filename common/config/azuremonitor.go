package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	AzureMonitorConnectionString   = "AZURE_MONITOR_CONNECTION_STRING"
	AzureMonitorInstrumentationKey = "AZURE_MONITOR_INSTRUMENTATION_KEY"
	AzureMonitorEndpoint           = "AZURE_MONITOR_ENDPOINT"
)

type AzureMonitor struct{}

func (a *AzureMonitor) DestType() common.DestinationType {
	return common.AzureMonitorDestinationType
}

func (a *AzureMonitor) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	cfg := dest.GetConfig()
	uniqueUri := "azuremonitor-" + dest.GetID()

	connectionString := cfg[AzureMonitorConnectionString]
	instrumentationKey := cfg[AzureMonitorInstrumentationKey]
	endpoint := cfg[AzureMonitorEndpoint]

	if connectionString == "" && instrumentationKey == "" && endpoint == "" {
		return nil, errorMissingKey(AzureMonitorConnectionString)
	}

	exporterName := "azuremonitor/" + uniqueUri
	currentConfig.Exporters[exporterName] = GenericMap{
		"connection_string":   connectionString,
		"instrumentation_key": instrumentationKey,
		"endpoint":            endpoint,
	}

	var pipelineNames []string

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/" + uniqueUri
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/" + uniqueUri
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
