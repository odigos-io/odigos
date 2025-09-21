package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
	semconv "go.opentelemetry.io/otel/semconv/v1.5.0"
)

// this processor should be added to any user telemetry pipeline to track the amount of data being exported by each source
const OdigosTrafficMetricsProcessorName = "odigostrafficmetrics"

const (
	// internal, not meant to be used outside of this service
	podNameProcessorName   = "resource/pod-name"
	ownMetricsExporterName = "otlp/odigos-own-telemetry-ui"
	ownMetricsReceiverName = "prometheus/self-metrics"
)

func calculateProcessorsConfigForOwnMetrics() config.GenericMap {
	processorsCfg := config.GenericMap{}

	processorsCfg[OdigosTrafficMetricsProcessorName] = config.GenericMap{
		// adding the following resource attributes to the metrics allows to aggregate the metrics by source.
		"res_attributes_keys": []string{
			string(semconv.ServiceNameKey),
			string(semconv.K8SNamespaceNameKey),
			string(semconv.K8SDeploymentNameKey),
			string(semconv.K8SStatefulSetNameKey),
			string(semconv.K8SDaemonSetNameKey),
		},
	}
	// this processor is used to add the pod name to the own metrics in the own metrics pipeline
	processorsCfg[podNameProcessorName] = config.GenericMap{
		"attributes": []config.GenericMap{{
			"key":    string(semconv.K8SPodNameKey),
			"value":  "${POD_NAME}",
			"action": "upsert",
		}},
	}

	return processorsCfg
}

func calculateReceiversConfigForOwnMetrics(ownMetricsPort int32) config.GenericMap {
	receiversCfg := config.GenericMap{}

	receiversCfg[ownMetricsReceiverName] = config.GenericMap{
		"config": config.GenericMap{
			"scrape_configs": []config.GenericMap{
				{
					"job_name":        "otelcol",
					"scrape_interval": "10s",
					"static_configs": []config.GenericMap{
						{
							"targets": []string{fmt.Sprintf("127.0.0.1:%d", ownMetricsPort)},
						},
					},
					"metric_relabel_configs": []config.GenericMap{
						{
							"source_labels": []string{"__name__"},
							"regex":         "(.*odigos.*)",
							"action":        "keep",
						},
					},
				},
			},
		},
	}

	return receiversCfg
}

func calculateExportersConfigForOwnMetrics(odigosNamespace string) config.GenericMap {
	exportersCfg := config.GenericMap{}

	endpoint := fmt.Sprintf("ui.%s:%d", odigosNamespace, consts.OTLPPort)
	exportersCfg[ownMetricsExporterName] = config.GenericMap{
		"endpoint": endpoint,
		"retry_on_failure": config.GenericMap{
			"enabled": false,
		},
		"tls": config.GenericMap{
			"insecure": true,
		},
	}

	return exportersCfg
}

func calculatePipelinesConfigForOwnMetrics() map[string]config.Pipeline {
	return map[string]config.Pipeline{
		"metrics/otelcol": config.Pipeline{
			Receivers:  []string{ownMetricsReceiverName},
			Processors: []string{podNameProcessorName},
			Exporters:  []string{ownMetricsExporterName},
		},
	}
}

func calculateServiceTelemetryConfigForOwnMetrics(ownMetricsPort int32) config.Telemetry {
	return config.Telemetry{
		Metrics: config.GenericMap{
			"readers": []config.GenericMap{
				{
					"pull": config.GenericMap{
						"exporter": config.GenericMap{
							"prometheus": config.GenericMap{
								"host": "0.0.0.0",
								"port": ownMetricsPort,
							},
						},
					},
				},
			},
		},
		Resource: map[string]*string{
			// The collector add "otelcol" as a service name, so we need to remove it
			// to avoid duplication, since we are interested in the instrumented services.
			string(semconv.ServiceNameKey): nil,
			// The collector adds its own version as a service version, which is not needed currently.
			string(semconv.ServiceVersionKey): nil,
		},
	}
}

// returns the collector config part that is needed for the collector own metrics pipeline
// merge it with other configs to get the full collector config
// Notice: this config part requires that you add the OdigosTrafficMetricsProcessorName processor
// to any pipeline for user telemetry to work correctly
func CalculateOwnMetricsConfig(ownMetricsPort int32, odigosNamespace string) config.Config {
	return config.Config{
		Receivers:  calculateReceiversConfigForOwnMetrics(ownMetricsPort),
		Exporters:  calculateExportersConfigForOwnMetrics(odigosNamespace),
		Processors: calculateProcessorsConfigForOwnMetrics(),
		Service: config.Service{
			Pipelines: calculatePipelinesConfigForOwnMetrics(),
			Telemetry: calculateServiceTelemetryConfigForOwnMetrics(ownMetricsPort),
		},
	}
}
