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
	if url, exists := dest.Spec.Data[genericOtlpUrlKey]; exists {
		genericOtlpExporterName := "otlp/generic-" + dest.Name
		currentConfig.Exporters[genericOtlpExporterName] = commonconf.GenericMap{
			"endpoint": url,
			"tls": commonconf.GenericMap{
				"insecure": true,
			},
		}
		if isTracingEnabled(dest) {
			tracesPipelineName := "traces/generic-" + dest.Name
			currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{genericOtlpExporterName},
			}
		}

		if isMetricsEnabled(dest) {
			metricsPipelineName := "metrics/generic-" + dest.Name
			currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{genericOtlpExporterName},
			}
		}

		if isLoggingEnabled(dest) {
			logsPipelineName := "logs/generic-" + dest.Name
			currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
				Exporters:  []string{genericOtlpExporterName},
			}
		}
	}
}
