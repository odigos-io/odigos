package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/config"
	semconv "go.opentelemetry.io/otel/semconv/v1.5.0"
)

const (
	odigletMetricsReceiverName = "prometheus/odiglet-metrics"
	odigletMetricsExporterName = "otlp_http/odiglet-metrics-out"
	odigletMetricsPipelineName = "metrics/odiglet-metrics"

	// Adds "odiglet" as a pod name so we can filter out the odiglet's own metrics in the UI
	odigletMetricsPodNameProcessorName = "resource/odiglet-pod-name"

	// keep only the eBPF instrumentation counters (java, python, nodejs) exposed by the odiglet.
	ebpfInstrumentationMetricsRegexPattern = "odigos_(java|python|nodejs)_ebpf_instrumentation_.*"

	// the cluster collector's own-metrics OTLP http receiver listens on this port.
	clusterCollectorOwnMetricsOtlpHttpPort = 44318
)

func odigletMetricsReceiverConfig(odigosNamespace string) config.GenericMap {
	odigletMetricsTarget := fmt.Sprintf("%s.%s:%d",
		k8sconsts.OdigletLocalTrafficServiceName,
		odigosNamespace,
		k8sconsts.OdigletMetricsServerPort,
	)

	return config.GenericMap{
		odigletMetricsReceiverName: config.GenericMap{
			"config": config.GenericMap{
				"scrape_configs": []config.GenericMap{
					{
						"job_name":           "odiglet",
						"scrape_interval":    "10s",
						"enable_compression": false,
						"static_configs": []config.GenericMap{
							{
								"targets": []string{odigletMetricsTarget},
							},
						},
						"metric_relabel_configs": []config.GenericMap{
							{
								"source_labels": []string{"__name__"},
								"regex":         ebpfInstrumentationMetricsRegexPattern,
								"action":        "keep",
							},
						},
					},
				},
			},
		},
	}
}

func odigletMetricsExporterConfig(odigosNamespace string) config.GenericMap {
	endpoint := fmt.Sprintf("http://%s.%s:%d",
		k8sconsts.OdigosClusterCollectorServiceName,
		odigosNamespace,
		clusterCollectorOwnMetricsOtlpHttpPort,
	)

	return config.GenericMap{
		odigletMetricsExporterName: config.GenericMap{
			"endpoint": endpoint,
			"retry_on_failure": config.GenericMap{
				"enabled": false,
			},
			"tls": config.GenericMap{
				"insecure": true,
			},
		},
	}
}

func odigletMetricsProcessorConfig() config.GenericMap {
	return config.GenericMap{
		odigletMetricsPodNameProcessorName: config.GenericMap{
			"attributes": []config.GenericMap{{
				"key":    string(semconv.K8SPodNameKey),
				"value":  "${POD_NAME}",
				"action": "upsert",
			}},
		},
	}
}

func odigletMetricsPipeline() map[string]config.Pipeline {
	return map[string]config.Pipeline{
		odigletMetricsPipelineName: {
			Receivers:  []string{odigletMetricsReceiverName},
			Processors: []string{odigletMetricsPodNameProcessorName},
			Exporters:  []string{odigletMetricsExporterName},
		},
	}
}

// Assembles a metrics config to be added to the node collector, which scrapes the odiglet for metrics
func OdigletMetricsConfig(odigosNamespace string) config.Config {
	return config.Config{
		Receivers:  odigletMetricsReceiverConfig(odigosNamespace),
		Exporters:  odigletMetricsExporterConfig(odigosNamespace),
		Processors: odigletMetricsProcessorConfig(),
		Service: config.Service{
			Pipelines: odigletMetricsPipeline(),
		},
	}
}
