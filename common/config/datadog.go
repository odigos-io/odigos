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

func (d *Datadog) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	if !isTracingEnabled(dest) && !isLoggingEnabled(dest) && !isMetricsEnabled(dest) {
		return errors.New("Datadog destination does not have any signals to export")
	}

	site, exists := dest.GetConfig()[datadogSiteKey]
	if !exists {
		return errors.New("Datadog site not specified, gateway will not be configured for Datadog")
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
	if isTracingEnabled(dest) && isMetricsEnabled(dest) {
		currentConfig.Connectors[connectorName] = struct{}{}
		connectorEnabled = true
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/datadog-" + dest.GetID()
		exporters := []string{exporterName}
		if connectorEnabled {
			exporters = append(exporters, connectorName)
		}

		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: exporters,
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/datadog-" + dest.GetID()
		var receivers []string
		if connectorEnabled {
			receivers = []string{connectorName}
		}
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Receivers: receivers,
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/datadog-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
