package collectorconfig

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	kubeletstatsReceiverName  = "kubeletstats"
	hostmetricsReceiverName   = "hostmetrics"
	odigosMetricsPipelineName = "metrics"
)

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

func MetricsConfig(nodeCG *odigosv1.CollectorsGroup, odigosNamespace string, manifestProcessorNames []string, metricsConfigSettings *odigosv1.CollectorsGroupMetricsCollectionSettings) config.Config {

	metricsPipelineProcessors := append([]string{
		batchProcessorName,         // always start with batch
		memoryLimiterProcessorName, // consider removing this for metrics, as they have footprint anyway
		nodeNameProcessorName,
		resourceDetectionProcessorName,
	}, manifestProcessorNames...)
	metricsPipelineProcessors = append(metricsPipelineProcessors, odigosTrafficMetricsProcessorName) // keep traffic metrics last for most accurate tracking

	receivers, pipelineReceiverNames := metricsReceivers(metricsConfigSettings)
	if len(pipelineReceiverNames) == 0 {
		// if all metrics sources are not enabled, skip the metrics pipeline generation as it has no receivers and will fail the collector
		return config.Config{}
	}

	return config.Config{
		Receivers: receivers,
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
