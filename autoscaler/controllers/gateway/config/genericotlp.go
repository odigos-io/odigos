package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	genericOtlpUrlKey = "OTLP_URL"
)

type GenericOTLP struct{}

func (g *GenericOTLP) DestType() common.DestinationType {
	return common.GenericOTLPDestinationType
}

func (g *GenericOTLP) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if url, exists := dest.Spec.Data[jaegerUrlKey]; exists {
		genericOtlpExporterName := "otlp/generic"
		currentConfig.Exporters[genericOtlpExporterName] = commonconf.GenericMap{
			"endpoint": url,
		}
		if isTracingEnabled(dest) {
			currentConfig.Service.Pipelines["traces/generic"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{genericOtlpExporterName},
			}
		}

		if isMetricsEnabled(dest) {
			currentConfig.Service.Pipelines["metrics/generic"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{genericOtlpExporterName},
			}
		}

		if isLoggingEnabled(dest) {
			currentConfig.Service.Pipelines["logs/generic"] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{genericOtlpExporterName},
			}
		}
	}
}
