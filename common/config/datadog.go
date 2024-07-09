package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
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
		"hostname": consts.OdigosClusterCollectorDeploymentName,
		"api": GenericMap{
			"key":  "${DATADOG_API_KEY}",
			"site": site,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/datadog-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/datadog-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
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
