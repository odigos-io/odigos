package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	datadogSiteKey = "DATADOG_SITE"
)

type Datadog struct{}

func (d *Datadog) DestType() common.DestinationType {
	return common.DatadogDestinationType
}

func (d *Datadog) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !IsTracingEnabled(dest) && !IsLoggingEnabled(dest) && !IsMetricsEnabled(dest) {
		return nil, errors.New("Datadog destination does not have any signals to export")
	}

	site, exists := dest.GetConfig()[datadogSiteKey]
	if !exists {
		return nil, errors.New("Datadog site not specified, gateway will not be configured for Datadog")
	}

	exporterName := "datadog/" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"hostname": "odigos-gateway",
		"api": GenericMap{
			"key":  "${DATADOG_API_KEY}",
			"site": site,
		},
	}

	connectorEnabled := false
	connectorName := "datadog/connector-" + dest.GetID()
	var pipelineNames []string
	if IsTracingEnabled(dest) && IsMetricsEnabled(dest) {
		currentConfig.Connectors[connectorName] = struct{}{}
		connectorEnabled = true
	}

	if IsTracingEnabled(dest) {
		tracesPipelineName := "traces/datadog-" + dest.GetID()
		exporters := []string{exporterName}
		if connectorEnabled {
			exporters = append(exporters, connectorName)
		}

		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: exporters,
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if IsMetricsEnabled(dest) {
		metricsPipelineName := "metrics/datadog-" + dest.GetID()
		var receivers []string
		if connectorEnabled {
			receivers = []string{connectorName}
		}
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Receivers: receivers,
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if IsLoggingEnabled(dest) {
		logsPipelineName := "logs/datadog-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
