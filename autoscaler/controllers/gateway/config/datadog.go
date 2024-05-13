package config

import (
	"errors"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

const (
	datadogSiteKey = "DATADOG_SITE"
)

type Datadog struct{}

func (d *Datadog) DestType() common.DestinationType {
	return common.DatadogDestinationType
}

func (d *Datadog) ModifyConfig(dest common.ExporterConfigurer, currentConfig *commonconf.Config) error {
	if !isTracingEnabled(dest) && !isLoggingEnabled(dest) && !isMetricsEnabled(dest) {
		return errors.New("Datadog destination does not have any signals to export")
	}

	site, exists := dest.GetConfig()[datadogSiteKey]
	if !exists {
		return errors.New("Datadog site not specified, gateway will not be configured for Datadog")
	}

	exporterName := "datadog/" + dest.GetName()
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"hostname": "odigos-gateway",
		"api": commonconf.GenericMap{
			"key":  "${DATADOG_API_KEY}",
			"site": site,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/datadog-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/datadog-" + dest.GetName()
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/datadog-" + dest.GetName()
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}
