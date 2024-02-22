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

	if isTracingEnabled(dest) {
		exporterName := "googlecloud/" + dest.Name
		currentConfig.Exporters[exporterName] = struct{}{}

		tracesPipelineName := "traces/googlecloud-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		exporterName := "googlecloud/" + dest.Name
		currentConfig.Exporters[exporterName] = commonconf.GenericMap{
			"log": commonconf.GenericMap{
				"default_log_name": "opentelemetry.io/collector-exported-log",
			},
		}

		logsPipelineName := "logs/googlecloud-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}
}
