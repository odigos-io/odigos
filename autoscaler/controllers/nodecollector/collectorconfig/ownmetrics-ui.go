package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// internal, not meant to be used outside of this service
const (

	// this processor should be added to any user telemetry pipeline to track the amount of data being exported by each source
	odigosTrafficMetricsProcessorName = "odigostrafficmetrics"

	podNameProcessorName       = "resource/pod-name"
	collectorRoleProcessorName = "resource/odigos-collector-role"
	ownMetricsUiExporterName   = "otlp/odigos-own-telemetry-ui"
	ownMetricsUiReceiverName   = "prometheus/self-metrics-ui"
	ownMetricsUiPipelineName   = "metrics/own-metrics-ui"
)

var staticOwnMetricsUiProcessors config.GenericMap
var uiOtlpEndpoint string

func init() {

	odigosNamespace := env.GetCurrentNamespace()

	staticOwnMetricsUiProcessors = config.GenericMap{
		odigosTrafficMetricsProcessorName: config.GenericMap{
			"res_attributes_keys": []string{
				string(semconv.ServiceNameKey),
				string(semconv.K8SNamespaceNameKey),
				string(semconv.K8SDeploymentNameKey),
				string(semconv.K8SStatefulSetNameKey),
				string(semconv.K8SDaemonSetNameKey),
				string(semconv.K8SCronJobNameKey),
				// Custom attribute for Argo Rollouts (no semconv key available)
				k8sconsts.K8SArgoRolloutNameAttribute,
				// Custom attribute to distinguish workload types that share the same semconv key (e.g., DeploymentConfig uses k8s.deployment.name)
				// This allows the UI to distinguish between DeploymentConfig and Deployment, and construct the correct Source workload.
				// Since DeploymentConfig uses k8s.deployment.name as the semconv key, we need to add this attribute to the list of attributes to be collected.
				consts.OdigosWorkloadKindAttribute,
				consts.OdigosWorkloadNameAttribute,
			},
		},
		podNameProcessorName: config.GenericMap{
			"attributes": []config.GenericMap{{
				"key":    string(semconv.K8SPodNameKey),
				"value":  "${POD_NAME}",
				"action": "upsert",
			}},
		},
		collectorRoleProcessorName: config.GenericMap{
			"attributes": []config.GenericMap{{
				"key":    "odigos.collector.role",
				"value":  string(k8sconsts.CollectorsRoleNodeCollector),
				"action": "upsert",
			}},
		},
	}

	uiOtlpEndpoint = fmt.Sprintf("ui.%s:%d", odigosNamespace, consts.OTLPPort)
}

func receiversConfigForOwnMetricsUi(ownMetricsPort int32) config.GenericMap {
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

	return receiversCfg
}

func serviceTelemetryConfigForOwnMetricsUi(ownMetricsPort int32) config.Telemetry {

	reader := config.GenericMap{
		"pull": config.GenericMap{
			"exporter": config.GenericMap{
				"prometheus": config.GenericMap{
					"host": "0.0.0.0",
					"port": ownMetricsPort,
				},
			},
		},
	}

	podNameFromEnv := "${POD_NAME}"
	nodeNameFromEnv := "${NODE_NAME}"
	return config.Telemetry{
		Metrics: config.MetricsConfig{
			Level:   "detailed",
			Readers: []config.GenericMap{reader},
		},
		Resource: map[string]*string{
			// The collector add "otelcol" as a service name, so we need to remove it
			// to avoid duplication, since we are interested in the instrumented services.
			string(semconv.ServiceNameKey): nil,
			// The collector adds its own version as a service version, which is not needed currently.
			string(semconv.ServiceVersionKey): nil,
			string(semconv.K8SPodNameKey):     &podNameFromEnv,
			string(semconv.K8SNodeNameKey):    &nodeNameFromEnv,
		},
	}
}

func ownMetricsExportersUi() config.GenericMap {
	return config.GenericMap{
		ownMetricsUiExporterName: config.GenericMap{
			"endpoint": uiOtlpEndpoint,
			"retry_on_failure": config.GenericMap{
				"enabled": false,
			},
			"tls": config.GenericMap{
				"insecure": true,
			},
		},
	}
}

func ownMetricsPipelinesUi() map[string]config.Pipeline {
	return map[string]config.Pipeline{
		ownMetricsUiPipelineName: {
			Receivers:  []string{ownMetricsUiReceiverName},
			Processors: []string{podNameProcessorName, collectorRoleProcessorName},
			Exporters:  []string{ownMetricsUiExporterName},
		},
	}
}

func OwnMetricsConfigUi(ownMetricsPort int32) config.Config {

	return config.Config{
		Receivers:  receiversConfigForOwnMetricsUi(ownMetricsPort),
		Exporters:  ownMetricsExportersUi(),
		Processors: staticOwnMetricsUiProcessors,
		Service: config.Service{
			Pipelines: ownMetricsPipelinesUi(),
			Telemetry: serviceTelemetryConfigForOwnMetricsUi(ownMetricsPort),
		},
	}
}
