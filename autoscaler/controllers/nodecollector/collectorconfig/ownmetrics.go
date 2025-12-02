package collectorconfig

import (
	"fmt"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// internal, not meant to be used outside of this service
const (
	odigosOwnTelemetryOtlpReceiverName = "otlp/odigos-own-metrics-in"
	ownMetricsStorePipelineName        = "metrics/own-metrics"
	odigosVictoriametricsExporterName  = "otlphttp/odigos-victoriametrics"
)

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

	// convert interval as duration string to milliseconds
	duration, err := time.ParseDuration(ownMetricsConfig.Interval)
	if err != nil {
		log.Log.Error(err, "failed to parse own metrics interval", "interval", ownMetricsConfig.Interval)
		return config.Telemetry{}
	}
	intervalMs := int64(duration.Milliseconds())

	reader := config.GenericMap{
		"periodic": config.GenericMap{
			"interval": intervalMs,
			"exporter": config.GenericMap{
				"otlp": config.GenericMap{
					"endpoint": "http://localhost:44318",
					"insecure": true,
					"protocol": "http/protobuf",
					"timeout":  "1s",
					"retry_on_failure": config.GenericMap{
						"enabled": false,
					},
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

func ownMetricsExporters(ownMetricsConfig *odigosv1.OdigosOwnMetricsSettings, odigosNamespace string) config.GenericMap {
	odigosVictoriametricsOtlpHttpEndpoint := fmt.Sprintf("http://odigos-victoriametrics.%s:8428/opentelemetry", odigosNamespace)
	if ownMetricsConfig.SendToOdigosMetricsStore {
		return config.GenericMap{
			odigosVictoriametricsExporterName: config.GenericMap{
				"endpoint": odigosVictoriametricsOtlpHttpEndpoint,
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
		ownMetricsStorePipelineName: config.Pipeline{
			Receivers: []string{odigosOwnTelemetryOtlpReceiverName},
			Exporters: []string{odigosVictoriametricsExporterName},
		},
	}
}

func OwnMetricsConfigPrometheus(ownMetricsConfig *odigosv1.OdigosOwnMetricsSettings, odigosNamespace string) (config.Config, []string) {

	var additionalMetricsReceivers []string
	if ownMetricsConfig.SendToMetricsDestinations {
		additionalMetricsReceivers = append(additionalMetricsReceivers, odigosOwnTelemetryOtlpReceiverName)
	}

	return config.Config{
		Receivers: receiversConfigForOwnMetricsPrometheus(),
		Exporters: ownMetricsExporters(ownMetricsConfig, odigosNamespace),
		Service: config.Service{
			Pipelines: ownMetricsPipelines(ownMetricsConfig),
			Telemetry: serviceTelemetryConfigForOwnMetrics(ownMetricsConfig),
		},
	}, additionalMetricsReceivers
}
