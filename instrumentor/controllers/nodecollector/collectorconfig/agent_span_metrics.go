package collectorconfig

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

// The agents' server-span duration histograms, recorded before head sampling, go to odigos
// insights for trace surges. cumulativetodelta runs before attributes are dropped, while each
// process is still its own series; span.name stays bounded by the agents' overflow series.
const (
	AgentSpanMetricsInsightsPipelineName = "metrics/agent-span-metrics-insights"
	agentSpanMetricsFilterName           = "filter/agent-span-metrics"
	agentSpanMetricsDeltaName            = "cumulativetodelta/agent-span-metrics"
	agentSpanMetricsIdleName             = "filter/agent-span-metrics-idle"
	agentSpanMetricsAttrsName            = "transform/agent-span-metrics"
	agentSpanMetricsBatchName            = "batch/agent-span-metrics"
	agentSpanMetricsExporterName         = "otlp_grpc/agent-span-metrics-insights"
	agentSpanMetricsDurationName         = "traces.span.metrics.duration"
)

// AgentSpanMetricsToInsights reports whether agents record span metrics for trace surges.
func AgentSpanMetricsToInsights(nodeCG *odigosv1.CollectorsGroup) bool {
	return nodeCG != nil && nodeCG.Spec.Metrics != nil && nodeCG.Spec.Metrics.AgentSpanMetrics != nil &&
		nodeCG.Spec.Metrics.AgentSpanMetrics.Insights
}

func AgentSpanMetricsInsightsConfig(odigosNamespace string) config.Config {
	return config.Config{
		Processors: config.GenericMap{
			agentSpanMetricsFilterName: config.GenericMap{
				"error_mode": "ignore",
				"metrics": config.GenericMap{
					"metric":    []string{`name != "` + agentSpanMetricsDurationName + `"`},
					"datapoint": []string{`attributes["span.kind"] != "SPAN_KIND_SERVER"`},
				},
			},
			agentSpanMetricsDeltaName: config.GenericMap{
				"include": config.GenericMap{
					"metrics":    []string{agentSpanMetricsDurationName},
					"match_type": "strict",
				},
				// a process that exited leaves its series; forget them.
				"max_staleness": "10m",
			},
			agentSpanMetricsIdleName: config.GenericMap{
				"error_mode": "ignore",
				"metrics": config.GenericMap{
					"datapoint": []string{`count == 0`},
				},
			},
			agentSpanMetricsAttrsName: config.GenericMap{
				"error_mode": "ignore",
				"metric_statements": []config.GenericMap{
					{
						"context": "resource",
						"statements": []string{
							`keep_keys(attributes, ["k8s.namespace.name", "odigos.workload.kind", "odigos.workload.name", "service.name"])`,
						},
					},
					{
						"context": "datapoint",
						"statements": []string{
							`keep_keys(attributes, ["span.name", "span.kind", "status.code"])`,
						},
					},
				},
			},
			// the agents of a node export every few seconds each, out of step: a batch holds most of a round,
			// and at most 2000 data points, so a dense node's export stays well under the 4 MiB of a gRPC
			// message. send_batch_size may not exceed send_batch_max_size.
			agentSpanMetricsBatchName: config.GenericMap{
				"timeout":             "5s",
				"send_batch_size":     2000,
				"send_batch_max_size": 2000,
			},
		},
		Exporters: config.GenericMap{
			agentSpanMetricsExporterName: config.GenericMap{
				"endpoint":      k8sconsts.InsightsOtlpGrpcDNSEndpoint(odigosNamespace),
				"balancer_name": "round_robin",
				"tls":           config.GenericMap{"insecure": true},
				"compression":   "gzip",
				"retry_on_failure": config.GenericMap{
					"enabled": false,
				},
			},
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				AgentSpanMetricsInsightsPipelineName: {
					Receivers: []string{OTLPInReceiverName},
					Processors: []string{
						agentSpanMetricsFilterName,
						agentSpanMetricsDeltaName,
						agentSpanMetricsIdleName,
						agentSpanMetricsAttrsName,
						agentSpanMetricsBatchName,
					},
					Exporters: []string{agentSpanMetricsExporterName},
				},
			},
		},
	}
}
