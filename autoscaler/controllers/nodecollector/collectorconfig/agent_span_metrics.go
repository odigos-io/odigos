package collectorconfig

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

// The agents record span metrics as each span ends, before head sampling. With insights on, the
// node collector sends them to odigos-insights, which stores them in its ClickHouse for trace
// surges to evaluate. They reach the node collector on its OTLP receiver with the agents' other
// telemetry, so they do not depend on a metrics destination.
//
// Trace surges evaluate each workload's server spans and each of its operations, so before
// sending, the node collector reduces the agents' series - one per process, span name, method and
// status code - to one per workload, span name and error status:
//   - only the server-span duration histogram is kept: its count is the calls.
//   - each process's cumulative series is converted to deltas while it is still a series of its own,
//     and deltas without calls are dropped, so idle operations cost nothing.
//   - every attribute but the workload's identity, span.name, span.kind and status.code is dropped.
//     span.name is bounded by the agents, which fold the span names past their series limit into
//     one overflow series.
//   - a few seconds of exports are batched, and their histograms are summed per workload, span
//     name and status.
const (
	AgentSpanMetricsInsightsPipelineName = "metrics/agent-span-metrics-insights"
	agentSpanMetricsFilterName           = "filter/agent-span-metrics"
	agentSpanMetricsDeltaName            = "cumulativetodelta/agent-span-metrics"
	agentSpanMetricsIdleName             = "filter/agent-span-metrics-idle"
	agentSpanMetricsAttrsName            = "transform/agent-span-metrics"
	agentSpanMetricsBatchName            = "batch/agent-span-metrics"
	agentSpanMetricsGroupName            = "groupbyattrs/agent-span-metrics"
	agentSpanMetricsAggregateName        = "metricstransform/agent-span-metrics"
	agentSpanMetricsExporterName         = "otlp_grpc/agent-span-metrics-insights"
	agentSpanMetricsDurationName         = "traces.span.metrics.duration"
)

// AgentSpanMetricsToInsights reports whether agents record span metrics for trace surges, which
// odigos insights stores.
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
			// the agents of a node export every few seconds each, out of step: a batch holds most of a round.
			agentSpanMetricsBatchName: config.GenericMap{
				"timeout": "5s",
			},
			// merges the resources the dropped attributes told apart: one per workload.
			agentSpanMetricsGroupName: config.GenericMap{},
			agentSpanMetricsAggregateName: config.GenericMap{
				"transforms": []config.GenericMap{
					{
						"include": agentSpanMetricsDurationName,
						"action":  "update",
						"operations": []config.GenericMap{
							{
								"action":           "aggregate_labels",
								"label_set":        []string{"span.name", "span.kind", "status.code"},
								"aggregation_type": "sum",
							},
						},
					},
				},
			},
		},
		Exporters: config.GenericMap{
			agentSpanMetricsExporterName: config.GenericMap{
				"endpoint":      k8sconsts.InsightsOtlpGrpcDNSEndpoint(odigosNamespace),
				"balancer_name": "round_robin",
				"tls":           config.GenericMap{"insecure": true},
				"compression":   "none",
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
						agentSpanMetricsGroupName,
						agentSpanMetricsAggregateName,
					},
					Exporters: []string{agentSpanMetricsExporterName},
				},
			},
		},
	}
}
