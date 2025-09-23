package collectorconfig

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	kubeletstatsReceiverName  = "kubeletstats"
	hostmetricsReceiverName   = "hostmetrics"
	odigosMetricsPipelineName = "metrics"
	spanMetricsConnectorName  = "spanmetrics"
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
			"collection_interval":  "10s",
		}
	}

	if metricsConfigSettings.HostMetrics != nil {
		pipelineReceiverNames = append(pipelineReceiverNames, hostmetricsReceiverName)
		receivers[hostmetricsReceiverName] = config.GenericMap{
			"collection_interval": "10s",
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

func metricsConnectors(metricsConfigSettings *odigosv1.CollectorsGroupMetricsCollectionSettings) (config.GenericMap, []string) {
	connectors := config.GenericMap{}
	connectorNamesToAdd := []string{}

	if metricsConfigSettings.SpanMetrics != nil {
		connectorNamesToAdd = append(connectorNamesToAdd, spanMetricsConnectorName)
		connectors[spanMetricsConnectorName] = config.GenericMap{
			"histogram": config.GenericMap{
				"explicit": config.GenericMap{
					"buckets": []string{"100us", "1ms", "2ms", "6ms", "10ms", "100ms", "250ms"},
				},
			},
			// Taking into account changes in the semantic conventions, to support a range of instrumentation libraries
			"dimensions": []config.GenericMap{
				{
					"name": "http.method",
				},
				{
					"name": "http.request.method",
				},
				{
					"name": "http.status_code",
				},
				{
					"name": "http.response.status_code",
				},
				{
					"name": "http.route",
				},
				{
					"name": "k8s.pod.name",
				},
				{
					"name": "k8s.namespace.name",
				},
				{
					"name": "k8s.container.name",
				},
			},
			"exemplars": config.GenericMap{
				"enabled": true,
			},
			"exclude_dimensions":              []string{"status.code"},
			"dimensions_cache_size":           1000,
			"aggregation_temporality":         "AGGREGATION_TEMPORALITY_CUMULATIVE",
			"metrics_flush_interval":          "15s",
			"metrics_expiration":              "5m",
			"resource_metrics_key_attributes": []string{"service.name", "telemetry.sdk.language", "telemetry.sdk.name"},
			"events": config.GenericMap{
				"enabled": true,
				"dimensions": []config.GenericMap{
					{
						"name": "exception.type",
					},
					{
						"name": "exception.message",
					},
				},
			},
		}
	}

	return connectors, connectorNamesToAdd
}

func MetricsConfig(nodeCG *odigosv1.CollectorsGroup, odigosNamespace string, manifestProcessorNames []string, metricsConfigSettings *odigosv1.CollectorsGroupMetricsCollectionSettings) config.Config {

	metricsPipelineProcessors := append([]string{
		BatchProcessorName,         // always start with batch
		MemoryLimiterProcessorName, // consider removing this for metrics, as they have footprint anyway
		NodeNameProcessorName,
		ResourceDetectionProcessorName,
	}, manifestProcessorNames...)
	metricsPipelineProcessors = append(metricsPipelineProcessors, OdigosTrafficMetricsProcessorName) // keep traffic metrics last for most accurate tracking

	receivers, pipelineReceiverNames := metricsReceivers(metricsConfigSettings)
	if len(pipelineReceiverNames) == 0 {
		// if all metrics sources are not enabled, skip the metrics pipeline generation as it has no receivers and will fail the collector
		return config.Config{}
	}

	// add connectors for span to metrics
	connectors, connectorNamesToAdd := metricsConnectors(metricsConfigSettings)

	pipelineExporterNames := append(connectorNamesToAdd, ClusterCollectorExporterName)

	return config.Config{
		Receivers:  receivers,
		Connectors: connectors,
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				odigosMetricsPipelineName: {
					Receivers:  pipelineReceiverNames,
					Processors: metricsPipelineProcessors,
					Exporters:  pipelineExporterNames,
				},
			},
		},
	}
}
