package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	semconv "go.opentelemetry.io/otel/semconv/v1.5.0"
)

// internal, not meant to be used outside of this service
const (

	// this processor should be added to any user telemetry pipeline to track the amount of data being exported by each source
	odigosTrafficMetricsProcessorName = "odigostrafficmetrics"

	podNameProcessorName             = "resource/pod-name"
	ownMetricsExporterName           = "otlp/odigos-own-telemetry-ui"
	ownMetricsUiReceiverName         = "prometheus/self-metrics-ui"
	ownMetricsPrometheusReceiverName = "prometheus/self-metrics-prometheus"
	ownMetricsUiPipelineName         = "metrics/own-metrics-ui"
	ownMetricsPrometheusPipelineName = "metrics/own-metrics-prometheus"
	odigosPrometheusExporterName     = "otlphttp/odigos-prometheus"
)

var staticOwnMetricsProcessors config.GenericMap
var uiOtlpEndpoint string
var odigosPrometheusOtlpHttpEndpoint string

func init() {

	odigosNamespace := env.GetCurrentNamespace()

	staticOwnMetricsProcessors = config.GenericMap{
		odigosTrafficMetricsProcessorName: config.GenericMap{
			"res_attributes_keys": []string{
				string(semconv.ServiceNameKey),
				string(semconv.K8SNamespaceNameKey),
				string(semconv.K8SDeploymentNameKey),
				string(semconv.K8SStatefulSetNameKey),
				string(semconv.K8SDaemonSetNameKey),
				string(semconv.K8SCronJobNameKey),
				// Custom attribute to distinguish workload types that share the same semconv key (e.g., DeploymentConfig uses k8s.deployment.name)
				// This allows the UI to distinguish between DeploymentConfig and Deployment, and construct the correct Source workload.
				// Since DeploymentConfig uses k8s.deployment.name as the semconv key, we need to add this attribute to the list of attributes to be collected.
				consts.OdigosWorkloadKindAttribute,
			},
		},
		podNameProcessorName: config.GenericMap{
			"attributes": []config.GenericMap{{
				"key":    string(semconv.K8SPodNameKey),
				"value":  "${POD_NAME}",
				"action": "upsert",
			}},
		},
	}

	uiOtlpEndpoint = fmt.Sprintf("ui.%s:%d", odigosNamespace, consts.OTLPPort)
	odigosPrometheusOtlpHttpEndpoint = fmt.Sprintf("http://odigos-prometheus.%s:9090/api/v1/otlp", odigosNamespace)
}

func receiversConfigForOwnMetrics(ownMetricsPort int32, odigosPrometheusEnabled bool) config.GenericMap {
	receiversCfg := config.GenericMap{}

	receiversCfg[ownMetricsUiReceiverName] = config.GenericMap{
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

	if odigosPrometheusEnabled {
		receiversCfg[ownMetricsPrometheusReceiverName] = config.GenericMap{
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
					},
				},
			},
		}
	}

	return receiversCfg
}

func serviceTelemetryConfigForOwnMetrics(ownMetricsPort int32) config.Telemetry {
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

func ownMetricsExporters(odigosPrometheusEnabled bool) config.GenericMap {
	exporters := config.GenericMap{
		ownMetricsExporterName: config.GenericMap{
			"endpoint": uiOtlpEndpoint,
			"retry_on_failure": config.GenericMap{
				"enabled": false,
			},
			"tls": config.GenericMap{
				"insecure": true,
			},
		},
	}
	if odigosPrometheusEnabled {
		exporters[odigosPrometheusExporterName] = config.GenericMap{
			"endpoint": odigosPrometheusOtlpHttpEndpoint,
			"retry_on_failure": config.GenericMap{
				"enabled": false,
			},
			"tls": config.GenericMap{
				"insecure": true,
			},
		}
	}
	return exporters
}

func ownMetricsPipelines(odigosPrometheusEnabled bool) map[string]config.Pipeline {
	pipelines := map[string]config.Pipeline{
		ownMetricsUiPipelineName: {
			Receivers:  []string{ownMetricsUiReceiverName},
			Processors: []string{podNameProcessorName},
			Exporters:  []string{ownMetricsExporterName},
		},
	}
	if odigosPrometheusEnabled {
		pipelines[ownMetricsPrometheusPipelineName] = config.Pipeline{
			Receivers:  []string{ownMetricsPrometheusReceiverName},
			Processors: []string{podNameProcessorName},
			Exporters:  []string{odigosPrometheusExporterName},
		}
	}
	return pipelines
}

// returns the collector config part that is needed for the collector own metrics pipeline
// merge it with other configs to get the full collector config
// Notice: this config part requires that you add the OdigosTrafficMetricsProcessorName processor
// to any pipeline for user telemetry to work correctly
func OwnMetricsConfig(ownMetricsPort int32, odigosPrometheusEnabled bool) config.Config {

	return config.Config{
		Receivers:  receiversConfigForOwnMetrics(ownMetricsPort, odigosPrometheusEnabled),
		Exporters:  ownMetricsExporters(odigosPrometheusEnabled),
		Processors: staticOwnMetricsProcessors,
		Service: config.Service{
			Pipelines: ownMetricsPipelines(odigosPrometheusEnabled),
			Telemetry: serviceTelemetryConfigForOwnMetrics(ownMetricsPort),
		},
	}
}
