package collectorconfig

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	kubeletstatsReceiverName  = "kubeletstats"
	hostmetricsReceiverName   = "hostmetrics"
	odigosMetricsPipelineName = "metrics"

	agentSpanMetricsOnlyPipelineName = "metrics/agent-span-metrics"
	dropAgentSpanMetricsName         = "filter/drop-agent-span-metrics"
	keepAgentSpanMetricsName         = "filter/keep-agent-span-metrics"
	agentSpanMetricsNamePattern      = `IsMatch(name, "^traces\\.span\\.metrics\\.")`
)

func metricsReceivers(metricsConfigSettings *odigosv1.CollectorsGroupMetricsCollectionSettings) (config.GenericMap, []string) {
	receivers := config.GenericMap{}
	pipelineReceiverNames := []string{}

	if metricsConfigSettings.AgentsTelemetry != nil {
		pipelineReceiverNames = append(pipelineReceiverNames, OTLPInReceiverName)
	}

	if metricsConfigSettings.KubeletStats != nil {
		pipelineReceiverNames = append(pipelineReceiverNames, kubeletstatsReceiverName)
		receivers[kubeletstatsReceiverName] = config.GenericMap{
			"auth_type":            "serviceAccount",
			"endpoint":             "https://${env:NODE_IP}:10250",
			"insecure_skip_verify": true,
			"collection_interval":  metricsConfigSettings.KubeletStats.Interval,
		}
	}

	if metricsConfigSettings.HostMetrics != nil {
		pipelineReceiverNames = append(pipelineReceiverNames, hostmetricsReceiverName)
		receivers[hostmetricsReceiverName] = config.GenericMap{
			"collection_interval": metricsConfigSettings.HostMetrics.Interval,
			"root_path":           "/hostfs",
			"scrapers": config.GenericMap{
				"paging": config.GenericMap{
					"metrics": config.GenericMap{
						"system.paging.utilization": config.GenericMap{
							"enabled": true,
						},
					},
				},
				"cpu": config.GenericMap{
					"metrics": config.GenericMap{
						"system.cpu.utilization": config.GenericMap{
							"enabled": true,
						},
					},
				},
				"disk": struct{}{},
				"filesystem": config.GenericMap{
					"metrics": config.GenericMap{
						"system.filesystem.utilization": config.GenericMap{
							"enabled": true,
						},
					},
					"exclude_mount_points": config.GenericMap{
						"match_type":   "regexp",
						"mount_points": []string{"/var/lib/kubelet/*"},
					},
				},
				"load":      struct{}{},
				"memory":    struct{}{},
				"network":   struct{}{},
				"processes": struct{}{},
			},
		}
	}

	return receivers, pipelineReceiverNames
}

type MetricsConfigOptions struct {
	CommonSignalConfig
	MetricsConfigSettings *odigosv1.CollectorsGroupMetricsCollectionSettings
}

func MetricsConfig(nodeCG *odigosv1.CollectorsGroup, opts MetricsConfigOptions) config.Config {

	baseProcessors := []string{
		batchProcessorName,         // always start with batch
		memoryLimiterProcessorName, // consider removing this for metrics, as they have footprint anyway
		nodeNameProcessorName,
	}
	if opts.ResourceDetectionEnabled {
		baseProcessors = append(baseProcessors, resourceDetectionProcessorName)
	}
	settings := opts.MetricsConfigSettings
	metricsPipelineProcessors := append([]string{}, baseProcessors...)
	metricsPipelineProcessors = append(metricsPipelineProcessors, opts.ManifestProcessorNames...)

	// Agents that record span metrics send them with the rest of their telemetry. Those trace surges
	// had them record go to odigos insights, and to the metrics destinations only when one wants span
	// metrics or span metrics in the agents are enabled.
	var processors config.GenericMap
	pipelines := map[string]config.Pipeline{}
	if agentSpanMetrics := settings.AgentSpanMetrics; agentSpanMetrics != nil {
		if !agentSpanMetrics.Destinations && settings.AgentsTelemetry != nil && settings.SpanMetrics == nil {
			processors = config.GenericMap{dropAgentSpanMetricsName: config.GenericMap{
				"error_mode": "ignore",
				"metrics":    config.GenericMap{"metric": []string{agentSpanMetricsNamePattern}},
			}}
			metricsPipelineProcessors = append(metricsPipelineProcessors, dropAgentSpanMetricsName)
		}
		if settings.AgentsTelemetry == nil && settings.SpanMetrics != nil {
			// the span metrics connector skips the spans these agents counted: their own span metrics
			// take their place.
			processors = config.GenericMap{keepAgentSpanMetricsName: config.GenericMap{
				"error_mode": "ignore",
				"metrics":    config.GenericMap{"metric": []string{"not " + agentSpanMetricsNamePattern}},
			}}
			pipelines[agentSpanMetricsOnlyPipelineName] = config.Pipeline{
				Receivers:  []string{OTLPInReceiverName},
				Processors: append(append([]string{}, baseProcessors...), keepAgentSpanMetricsName),
				Exporters:  []string{clusterCollectorMetricsExporterName},
			}
		}
	}
	metricsPipelineProcessors = append(metricsPipelineProcessors, odigosTrafficMetricsProcessorName) // keep traffic metrics last for most accurate tracking

	receivers, pipelineReceiverNames := metricsReceivers(settings)
	if len(pipelineReceiverNames) > 0 {
		pipelines[odigosMetricsPipelineName] = config.Pipeline{
			Receivers:  pipelineReceiverNames,
			Processors: metricsPipelineProcessors,
			Exporters:  []string{clusterCollectorMetricsExporterName},
		}
	}
	if len(pipelines) == 0 {
		// if all metrics sources are not enabled, skip the metrics pipeline generation as it has no receivers and will fail the collector
		return config.Config{}
	}

	return config.Config{
		Receivers:  receivers,
		Processors: processors,
		Service:    config.Service{Pipelines: pipelines},
	}
}
