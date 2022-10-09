package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

type NewRelic struct{}

func (n *NewRelic) DestType() common.DestinationType {
	return common.NewRelicDestinationType
}

func (n *NewRelic) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	currentConfig.Exporters["otlp/newrelic"] = commonconf.GenericMap{
		"endpoint": "https://otlp.nr-data.net:4317",
		"headers": commonconf.GenericMap{
			"api-key": "${NEWRELIC_API_KEY}",
		},
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/newrelic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/newrelic"},
		}
	}

	if isMetricsEnabled(dest) {
		currentConfig.Service.Pipelines["metrics/newrelic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/newrelic"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Service.Pipelines["logs/newrelic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/newrelic"},
		}
	}
}
