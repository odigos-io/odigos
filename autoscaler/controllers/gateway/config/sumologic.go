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
	currentConfig.Exporters["otlphttp/sumologic"] = commonconf.GenericMap{
		"endpoint": "${SUMOLOGIC_COLLECTION_URL}",
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/sumologic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/sumologic"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Service.Pipelines["logs/sumologic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/sumologic"},
		}
	}

	if isMetricsEnabled(dest) {
		currentConfig.Service.Pipelines["metrics/sumologic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/sumologic"},
		}
	}
}
