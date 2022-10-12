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
	if isTracingEnabled(dest) || isMetricsEnabled(dest) {
		site, exists := dest.Spec.Data[datadogSiteKey]
		if !exists {
			log.Log.V(0).Info("Datadog site not specified, gateway will not be configured for Datadog")
			return
		}

		currentConfig.Exporters["datadog"] = commonconf.GenericMap{
			"api": commonconf.GenericMap{
				"key":  "${DATADOG_API_KEY}",
				"site": site,
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
