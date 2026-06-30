package collectorconfig

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	kubeletstatsReceiverName  = "kubeletstats"
	hostmetricsReceiverName   = "hostmetrics"
	odigosMetricsPipelineName = "metrics"
	// obiMetricsRenameProcessorName renames OBI-produced metrics (e.g. network flow and TCP stats
	// metrics named "obi.*" / "obi_*") by replacing the "obi" prefix with "odigos", for consistency
	// and discoverability alongside other Odigos metrics in the platform.
	obiMetricsRenameProcessorName = "transform/odigos-obi-metrics-rename"
)

// obiMetricsRenameProcessorConfig returns a transform processor that replaces the "obi" name prefix of
// OBI metrics with "odigos". OBI emits metrics in dotted OTLP form (e.g. "obi.network.flow.bytes"),
// which becomes "odigos.network.flow.bytes"; the Prometheus underscore form ("obi_...") is handled
// defensively and becomes "odigos_...". The separator following the prefix is preserved by only
// replacing the leading "obi" token.
func obiMetricsRenameProcessorConfig() config.GenericMap {
	return config.GenericMap{
		"error_mode": "ignore",
		"metric_statements": []config.GenericMap{
			{
				"context": "metric",
				"statements": []string{
					`replace_pattern(name, "^obi", "odigos") where IsMatch(name, "^obi[._]")`,
				},
			},
		},
	}
}

func metricsReceivers(metricsConfigSettings *odigosv1.CollectorsGroupMetricsCollectionSettings) (config.GenericMap, []string) {
	receivers := config.GenericMap{}
	// For now, we are adding odigosebpfreceiver in any case if metrics enabled.
	pipelineReceiverNames := []string{odigosEbpfReceiverName}

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
	// NetworkMetricsEnabled indicates at least one workload on this node has OBI network metrics
	// enabled (via a networkMetrics InstrumentationRule). Only then is the OBI metric rename
	// processor added to the pipeline.
	NetworkMetricsEnabled bool
}

// AnyNetworkMetricsEnabled reports whether any container in the given sources has OBI network metrics
// enabled. It is used to decide whether the OBI metric rename processor is needed in the pipeline.
func AnyNetworkMetricsEnabled(sources *odigosv1.InstrumentationConfigList) bool {
	if sources == nil {
		return false
	}
	for i := range sources.Items {
		for j := range sources.Items[i].Spec.Containers {
			metrics := sources.Items[i].Spec.Containers[j].Metrics
			// Enablement is presence-based: a non-nil NetworkMetrics means metrics are collected.
			if metrics != nil && metrics.NetworkMetrics != nil {
				return true
			}
		}
	}
	return false
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
	metricsPipelineProcessors := baseProcessors
	// Normalize OBI metric names to the "odigos" prefix before user (manifest) processors and traffic
	// accounting, so downstream processors and destinations see the consistent Odigos naming. Only
	// added when a workload on this node has OBI network metrics enabled, since that's the only source
	// of "obi.*" metrics.
	extraProcessors := config.GenericMap{}
	if opts.NetworkMetricsEnabled {
		metricsPipelineProcessors = append(metricsPipelineProcessors, obiMetricsRenameProcessorName)
		extraProcessors[obiMetricsRenameProcessorName] = obiMetricsRenameProcessorConfig()
	}
	metricsPipelineProcessors = append(metricsPipelineProcessors, opts.ManifestProcessorNames...)
	metricsPipelineProcessors = append(metricsPipelineProcessors, odigosTrafficMetricsProcessorName) // keep traffic metrics last for most accurate tracking

	receivers, pipelineReceiverNames := metricsReceivers(opts.MetricsConfigSettings)
	if len(pipelineReceiverNames) == 0 {
		// if all metrics sources are not enabled, skip the metrics pipeline generation as it has no receivers and will fail the collector
		return config.Config{}
	}

	return config.Config{
		Receivers:  receivers,
		Processors: extraProcessors,
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				odigosMetricsPipelineName: {
					Receivers:  pipelineReceiverNames,
					Processors: metricsPipelineProcessors,
					Exporters:  []string{clusterCollectorMetricsExporterName},
				},
			},
		},
	}
}
