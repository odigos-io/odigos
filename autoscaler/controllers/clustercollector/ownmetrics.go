package clustercollector

import (
	"fmt"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	odigosOwnTelemetryOtlpReceiverName = "otlp/odigos-own-metrics-in"
	gatewayPrometheusReceiverName      = "prometheus/gateway-own-metrics"
	ownMetricsStorePipelineName        = "metrics/own-metrics"
	odigosVictoriametricsExporterName  = "otlphttp/odigos-victoriametrics"
	defaultOwnMetricsScrapeInterval    = "10s"
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

func gatewayPrometheusReceiverConfig(ownTelemetryPort int32, interval string) config.GenericMap {
	if interval == "" {
		interval = defaultOwnMetricsScrapeInterval
	}
	return config.GenericMap{
		"config": config.GenericMap{
			"scrape_configs": []config.GenericMap{
				{
					"job_name":        "otelcol-gateway",
					"scrape_interval": interval,
					"static_configs": []config.GenericMap{
						{
							"targets": []string{fmt.Sprintf("127.0.0.1:%d", ownTelemetryPort)},
						},
					},
				},
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

	receivers := []string{odigosOwnTelemetryOtlpReceiverName}
	exporters := []string{}

	if ownMetricsConfig.SendToOdigosMetricsStore {
		c.Exporters[odigosVictoriametricsExporterName] = victoriaMetricsExporter(odigosNamespace)
		exporters = append(exporters, odigosVictoriametricsExporterName)

		// Scrape the gateway's own prometheus endpoint so that gateway metrics
		// are also forwarded to the metrics store, using the configured interval.
		c.Receivers[gatewayPrometheusReceiverName] = gatewayPrometheusReceiverConfig(ownTelemetryPort, ownMetricsConfig.Interval)
		receivers = append(receivers, gatewayPrometheusReceiverName)
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
		Receivers: receivers,
		Exporters: exporters,
	}

	return nil
}
