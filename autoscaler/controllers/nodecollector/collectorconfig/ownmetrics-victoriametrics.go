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

	odigletJobName               = "odiglet"
	odigletMetricsScrapeInterval = "10s"

	// Stamps the data collection pod's own name as k8s.pod.name on the scraped odiglet metrics.
	// The scrape carries no per-pod identity, so without this the series from every node's data would aggregate to be the same.
	// The UI queries these metrics grouped and filtered by k8s.pod.name, matching against the data collection pod names.
	odigletMetricsPodNameProcessorName = "resource/odiglet-pod-name"

	// keep only the eBPF instrumentation counters (java, python, nodejs) exposed by the odiglet.
	ebpfInstrumentationMetricsRegexPattern = "odigos_(java|python|nodejs)_ebpf_instrumentation_.*"

	// keep the ebpf-core shared-buffer event counters (emitted by the go/java greatwall instrumentations for example)
	ebpfCoreEventsRegexPattern = "odigos_ebpf_events_(sent|send_failed)_.*"

	// the cluster collector's own-metrics OTLP http receiver listens on this port.
	clusterCollectorOwnMetricsOtlpHttpPort = 44318
)

func odigletMetricsReceiverConfig() config.GenericMap {
	// The data collection collector runs as a container in the odiglet pod, so it can scrape the odiglet's metrics endpoint over localhost
	odigletMetricsTarget := fmt.Sprintf("localhost:%d", k8sconsts.OdigletMetricsServerPort)

	keepRegex := fmt.Sprintf("%s|%s", ebpfInstrumentationMetricsRegexPattern, ebpfCoreEventsRegexPattern)

	return config.GenericMap{
		odigletMetricsReceiverName: config.GenericMap{
			"config": config.GenericMap{
				"scrape_configs": []config.GenericMap{
					{
						"job_name":           odigletJobName,
						"scrape_interval":    odigletMetricsScrapeInterval,
						"enable_compression": false,
						"static_configs": []config.GenericMap{
							{
								"targets": []string{odigletMetricsTarget},
							},
						},
						"metric_relabel_configs": []config.GenericMap{
							{
								"source_labels": []string{"__name__"},
								"regex":         keepRegex,
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
		Receivers:  odigletMetricsReceiverConfig(),
		Exporters:  odigletMetricsExporterConfig(odigosNamespace),
		Processors: odigletMetricsProcessorConfig(),
		Service: config.Service{
			Pipelines: odigletMetricsPipeline(),
		},
	}
}
