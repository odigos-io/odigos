package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
)

type Honeycomb struct{}

func (h *Honeycomb) DestType() odigosv1.DestinationType {
	return odigosv1.HoneycombDestinationType
}

func (h *Honeycomb) ModifyConfig(dest *odigosv1.Destination, currentConfig *Config) {
	if isTracingEnabled(dest) {
		currentConfig.Exporters["otlp/honeycomb"] = genericMap{
			"endpoint": "api.honeycomb.io:443",
			"headers": genericMap{
				"x-honeycomb-team": "${API_KEY}",
			},
		}

		currentConfig.Service.Pipelines["traces/honeycomb"] = Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/honeycomb"},
		}
	}
}
