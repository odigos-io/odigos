package clustercollector

import (
	"fmt"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	odigosOwnTelemetryOtlpReceiverName = "otlp/odigos-own-metrics-in"
	ownMetricsStorePipelineName        = "metrics/own-metrics"
	odigosVictoriametricsExporterName  = "otlphttp/odigos-victoriametrics"
)

func receiversConfigForOwnMetricsPrometheus() config.GenericMap {
	return config.GenericMap{
		"protocols": config.GenericMap{
			"http": config.GenericMap{
				"endpoint": "0.0.0.0:44318",
			},
		},
	}
}

func victoriaMetricsExporter(odigosNamespace string) config.GenericMap {
	endpoint := fmt.Sprintf("http://odigos-victoriametrics.%s:8428/opentelemetry", odigosNamespace)
	return config.GenericMap{
		"endpoint": endpoint,
		"retry_on_failure": config.GenericMap{
			"enabled": false,
		},
		"tls": config.GenericMap{
			"insecure": true,
		},
	}
}

// addOwnMetricsPipeline integrates own-metrics collection into the gateway config.
func addOwnMetricsPipeline(c *config.Config, ownMetricsConfig *odigosv1.OdigosOwnMetricsSettings, odigosNamespace string, ownTelemetryPort int32, destinationPipelineNames []string) error {
	c.Receivers[odigosOwnTelemetryOtlpReceiverName] = receiversConfigForOwnMetricsPrometheus()

	exporters := []string{}

	if ownMetricsConfig.SendToOdigosMetricsStore {
		c.Exporters[odigosVictoriametricsExporterName] = victoriaMetricsExporter(odigosNamespace)

		exporters = append(exporters, odigosVictoriametricsExporterName)
	}

	if ownMetricsConfig.SendToMetricsDestinations {
		for _, pipelineName := range destinationPipelineNames {
			if !strings.HasPrefix(pipelineName, "metrics/") {
				continue
			}
			pipeline := c.Service.Pipelines[pipelineName]
			exporters = append(exporters, pipeline.Exporters...)
		}
	}

	c.Service.Pipelines[ownMetricsStorePipelineName] = config.Pipeline{
		Receivers: []string{odigosOwnTelemetryOtlpReceiverName},
		Exporters: exporters,
	}

	return nil
}
