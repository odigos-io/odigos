package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
)

type Datadog struct{}

func (d *Datadog) DestType() odigosv1.DestinationType {
	return odigosv1.DatadogDestinationType
}

func (d *Datadog) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isTracingEnabled(dest) || isMetricsEnabled(dest) {
		currentConfig.Exporters["datadog"] = commonconf.GenericMap{
			"api": commonconf.GenericMap{
				"key":  "${API_KEY}",
				"site": dest.Spec.Data.Datadog.Site,
			},
		}
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/datadog"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"datadog"},
		}
	}

	if isMetricsEnabled(dest) {
		currentConfig.Service.Pipelines["metrics/datadog"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"datadog"},
		}
	}
}
