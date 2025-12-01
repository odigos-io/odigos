package collectorconfig

import (
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// internal, not meant to be used outside of this service
const (
	odigosOwnTelemetryOtlpReceiverName = "otlp/odigos-own-metrics-in"
	ownMetricsPrometheusPipelineName   = "metrics/own-metrics-prometheus"
	odigosPrometheusExporterName       = "otlphttp/odigos-prometheus"
)

var odigosPrometheusOtlpHttpEndpoint string

func init() {
	odigosNamespace := env.GetCurrentNamespace()
	odigosPrometheusOtlpHttpEndpoint = fmt.Sprintf("http://odigos-prometheus.%s:9090/api/v1/otlp", odigosNamespace)
}

func receiversConfigForOwnMetricsPrometheus() config.GenericMap {
	return config.GenericMap{
		odigosOwnTelemetryOtlpReceiverName: config.GenericMap{
			"protocols": config.GenericMap{
				"grpc": config.GenericMap{
					"endpoint": "0.0.0.0:44317",
				},
				"http": config.GenericMap{
					"endpoint": "0.0.0.0:44318",
				},
			},
		},
	}
}

func serviceTelemetryConfigForOwnMetrics(ownMetricsConfig *odigosv1.OdigosOwnMetricsSettings) config.Telemetry {

	reader := config.GenericMap{
		"periodic": config.GenericMap{
			"interval": ownMetricsConfig.Interval,
			"exporter": config.GenericMap{
				"otlp": config.GenericMap{
					"endpoint": "http://localhost:44318",
					"insecure": true,
					"protocol": "http/protobuf",
				},
			},
		},
	}

	return config.Telemetry{
		Metrics: config.MetricsConfig{
			Readers: []config.GenericMap{reader},
		},
	}
}

func ownMetricsExporters(ownMetricsConfig *odigosv1.OdigosOwnMetricsSettings) config.GenericMap {
	if ownMetricsConfig.SendToOdigosMetricsStore {
		return config.GenericMap{
			odigosPrometheusExporterName: config.GenericMap{
				"endpoint": odigosPrometheusOtlpHttpEndpoint,
				"retry_on_failure": config.GenericMap{
					"enabled": false,
				},
				"tls": config.GenericMap{
					"insecure": true,
				},
			},
		}
	}
	return config.GenericMap{}
}

func ownMetricsPipelines(ownMetricsConfig *odigosv1.OdigosOwnMetricsSettings) map[string]config.Pipeline {

	if !ownMetricsConfig.SendToOdigosMetricsStore {
		return map[string]config.Pipeline{}
	}

	return map[string]config.Pipeline{
		ownMetricsPrometheusPipelineName: config.Pipeline{
			Receivers: []string{odigosOwnTelemetryOtlpReceiverName},
			Exporters: []string{odigosPrometheusExporterName},
		},
	}
}

func OwnMetricsConfigPrometheus(ownMetricsConfig *odigosv1.OdigosOwnMetricsSettings) (config.Config, []string) {

	var additionalMetricsReceivers []string
	if ownMetricsConfig.SendToMetricsDestinations {
		additionalMetricsReceivers = append(additionalMetricsReceivers, odigosOwnTelemetryOtlpReceiverName)
	}

	return config.Config{
		Receivers: receiversConfigForOwnMetricsPrometheus(),
		Exporters: ownMetricsExporters(ownMetricsConfig),
		Service: config.Service{
			Pipelines: ownMetricsPipelines(ownMetricsConfig),
			Telemetry: serviceTelemetryConfigForOwnMetrics(ownMetricsConfig),
		},
	}, additionalMetricsReceivers
}
