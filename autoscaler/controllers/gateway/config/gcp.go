package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

type GoogleCloud struct{}

func (g *GoogleCloud) DestType() common.DestinationType {
	return common.GoogleCloudDestinationType
}

func (g *GoogleCloud) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if isLoggingEnabled(dest) {
		currentConfig.Exporters["googlecloud"] = commonconf.GenericMap{
			"log": commonconf.GenericMap{
				"default_log_name": "opentelemetry.io/collector-exported-log",
			},
		}
	} else if isTracingEnabled(dest) {
		currentConfig.Exporters["googlecloud"] = struct{}{}
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/googlecloud"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"googlecloud"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Service.Pipelines["logs/googlecloud"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"googlecloud"},
		}
	}
}
