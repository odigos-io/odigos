package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	datadogSiteKey = "DATADOG_SITE"
)

type Datadog struct{}

func (d *Datadog) DestType() common.DestinationType {
	return common.DatadogDestinationType
}

func (d *Datadog) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if !isTracingEnabled(dest) && !isLoggingEnabled(dest) && !isMetricsEnabled(dest) {
		log.Log.V(0).Info("Datadog destination does not have any signals to export")
		return
	}

	site, exists := dest.Spec.Data[datadogSiteKey]
	if !exists {
		log.Log.V(0).Info("Datadog site not specified, gateway will not be configured for Datadog")
		return
	}

	exporterName := "datadog/" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"hostname": "odigos-gateway",
		"api": commonconf.GenericMap{
			"key":  "${DATADOG_API_KEY}",
			"site": site,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/datadog-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/datadog-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/datadog-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{exporterName},
		}
	}
}
