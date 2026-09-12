package collectorconfig

import (
	"fmt"
	"slices"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	kubeletstatsReceiverName  = "kubeletstats"
	hostmetricsReceiverName   = "hostmetrics"
	odigosMetricsPipelineName = "metrics"

	// JVM runtime metrics from the eBPF Java agent arrive over OTLP on otlp/in,
	// which joins the metrics pipeline only with agentsTelemetry. They are
	// reported whatever that setting says, so without it they get a route of
	// their own.
	jvmRuntimeMetricsPipelineName = "metrics/jvm-runtime"
	jvmRuntimeMetricsFilterName   = "filter/jvm-runtime"
	// Must equal ebpf-java-instrumentation's jvmmetrics.ScopeName: renaming the
	// scope there silently filters these metrics out.
	jvmRuntimeMetricsScopeName = "jvm-ebpf-metrics"
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
	// keep traffic metrics last for most accurate tracking
	pipelineProcessors := func(extra ...string) []string {
		processors := append(slices.Clone(baseProcessors), extra...)
		processors = append(processors, opts.ManifestProcessorNames...)
		return append(processors, odigosTrafficMetricsProcessorName)
	}

	receivers, pipelineReceiverNames := metricsReceivers(opts.MetricsConfigSettings)

	pipelines := map[string]config.Pipeline{}
	if len(pipelineReceiverNames) > 0 {
		pipelines[odigosMetricsPipelineName] = config.Pipeline{
			Receivers:  pipelineReceiverNames,
			Processors: pipelineProcessors(),
			Exporters:  []string{clusterCollectorMetricsExporterName},
		}
	}

	// Without agentsTelemetry, otlp/in is not in the metrics pipeline. JVM runtime
	// metrics never depended on that setting, so they get a pipeline of their own
	// that admits their scope and nothing else - every other OTLP metric stays
	// opted out.
	var processors config.GenericMap
	if opts.Tier.IsEnterprise() && opts.MetricsConfigSettings.AgentsTelemetry == nil {
		processors = jvmRuntimeMetricsProcessorConfig()
		pipelines[jvmRuntimeMetricsPipelineName] = config.Pipeline{
			Receivers:  []string{OTLPInReceiverName},
			Processors: pipelineProcessors(jvmRuntimeMetricsFilterName),
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
		Service: config.Service{
			Pipelines: pipelines,
		},
	}
}

// jvmRuntimeMetricsProcessorConfig drops every metric outside the eBPF Java
// agent's JVM runtime scope.
func jvmRuntimeMetricsProcessorConfig() config.GenericMap {
	return config.GenericMap{
		jvmRuntimeMetricsFilterName: config.GenericMap{
			"error_mode": "ignore",
			"metrics": config.GenericMap{
				"metric": []string{fmt.Sprintf("instrumentation_scope.name != %q", jvmRuntimeMetricsScopeName)},
			},
		},
	}
}
