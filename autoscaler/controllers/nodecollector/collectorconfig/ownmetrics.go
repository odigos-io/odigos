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

	podNameProcessorName   = "resource/pod-name"
	ownMetricsExporterName = "otlp/odigos-own-telemetry-ui"
	ownMetricsReceiverName = "prometheus/self-metrics"
	ownMetricsPipelineName = "metrics/own-metrics"
)

var staticOwnMetricsProcessors config.GenericMap
var staticOwnMetricsExporters config.GenericMap
var staticOwnMetricsPipelines map[string]config.Pipeline

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

	endpoint := fmt.Sprintf("ui.%s:%d", odigosNamespace, consts.OTLPPort)
	staticOwnMetricsExporters = config.GenericMap{
		ownMetricsExporterName: config.GenericMap{
			"endpoint": endpoint,
			"retry_on_failure": config.GenericMap{
				"enabled": false,
			},
			"tls": config.GenericMap{
				"insecure": true,
			},
		},
	}

	staticOwnMetricsPipelines = map[string]config.Pipeline{
		ownMetricsPipelineName: {
			Receivers:  []string{ownMetricsReceiverName},
			Processors: []string{podNameProcessorName},
			Exporters:  []string{ownMetricsExporterName},
		},
	}
}

func receiversConfigForOwnMetrics(ownMetricsPort int32) config.GenericMap {
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

// returns the collector config part that is needed for the collector own metrics pipeline
// merge it with other configs to get the full collector config
// Notice: this config part requires that you add the OdigosTrafficMetricsProcessorName processor
// to any pipeline for user telemetry to work correctly
func OwnMetricsConfig(ownMetricsPort int32) config.Config {
	return config.Config{
		Receivers:  receiversConfigForOwnMetrics(ownMetricsPort),
		Exporters:  staticOwnMetricsExporters,
		Processors: staticOwnMetricsProcessors,
		Service: config.Service{
			Pipelines: staticOwnMetricsPipelines,
			Telemetry: serviceTelemetryConfigForOwnMetrics(ownMetricsPort),
		},
	}
}
