package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
)

type Honeycomb struct{}

func (h *Honeycomb) DestType() odigosv1.DestinationType {
	return odigosv1.HoneycombDestinationType
}

func (h *Honeycomb) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isTracingEnabled(dest) {
		currentConfig.Exporters["otlp/honeycomb"] = commonconf.GenericMap{
			"endpoint": "api.honeycomb.io:443",
			"headers": commonconf.GenericMap{
				"x-honeycomb-team": "${API_KEY}",
			},
		}

		currentConfig.Service.Pipelines["traces/honeycomb"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/honeycomb"},
		}
	}
}
