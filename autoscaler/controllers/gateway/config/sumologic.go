package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

type SumoLogic struct{}

func (s *SumoLogic) DestType() common.DestinationType {
	return common.SumoLogicDestinationType
}

func (s *SumoLogic) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	exporterName := "otlphttp/sumologic-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": "${SUMOLOGIC_COLLECTION_URL}",
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/sumologic-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/sumologic-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/sumologic-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}
}
