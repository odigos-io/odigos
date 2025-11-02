package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	odigosEbpfReceiverName                = "odigosebpf"
	odigosTracesLoadbalancingExporterName = "loadbalancing/traces"
	odigosTracesPipelineName              = "traces"
)

var staticTracesReceivers config.GenericMap

func init() {
	staticTracesReceivers = config.GenericMap{
		odigosEbpfReceiverName: config.GenericMap{},
	}
}

func tracesExporters(nodeCG *odigosv1.CollectorsGroup, odigosNamespace string, tracesEnabledInClusterCollector bool, loadBalancingNeeded bool) (config.GenericMap, []string) {

	exporters := config.GenericMap{}
	exporterNames := []string{}

	// add exporter only if we are sending traces to the cluster collector
	if tracesEnabledInClusterCollector {

		// Add loadbalancing exporter for traces to ensure consistent gateway routing.
		// This needed for the service graph to work correctly and for the sampling actions to work correctly.
		// If load balancing is not needed, we use the common cluster collector exporter without load balancing.
		if loadBalancingNeeded {
			compression := "none"
			if nodeCG.Spec.OtlpExporterConfiguration != nil && nodeCG.Spec.OtlpExporterConfiguration.EnableDataCompression != nil && *nodeCG.Spec.OtlpExporterConfiguration.EnableDataCompression {
				compression = "gzip"
			}

			service := fmt.Sprintf("%s.%s", k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace)

			// Build the OTLP protocol configuration
			otlpConfig := config.GenericMap{
				"compression": compression,
				"tls": config.GenericMap{
					"insecure": true,
				},
			}

			// Add retry_on_failure configuration if present
			if nodeCG.Spec.OtlpExporterConfiguration != nil && nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure != nil {
				retryConfig := config.GenericMap{}

				// Only set enabled if not nil to avoid possible nil pointer dereference
				if nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure.Enabled != nil {
					retryConfig["enabled"] = *nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure.Enabled
				} else {
					// by default, retry on failure is enabled
					retryConfig["enabled"] = true
				}

				// Only add the interval fields if they are not empty
				if nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure.InitialInterval != "" {
					retryConfig["initial_interval"] = nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure.InitialInterval
				}
				if nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure.MaxInterval != "" {
					retryConfig["max_interval"] = nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure.MaxInterval
				}
				if nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure.MaxElapsedTime != "" {
					retryConfig["max_elapsed_time"] = nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure.MaxElapsedTime
				}

				otlpConfig["retry_on_failure"] = retryConfig
			}

			exporters[odigosTracesLoadbalancingExporterName] = config.GenericMap{
				"protocol": config.GenericMap{
					"otlp": otlpConfig,
				},
				"resolver": config.GenericMap{
					"k8s": config.GenericMap{
						"service": service,
					},
				},
			}
			exporterNames = append(exporterNames, odigosTracesLoadbalancingExporterName)
		} else {
			// Use the common cluster collector exporter
			// Note: The actual exporter merge by commonExporters before this function is called.
			// Here we just add it to the exporter name
			exporterNames = append(exporterNames, clusterCollectorTraceExporterName)
		}
	}

	return exporters, exporterNames
}

func TracesConfig(nodeCG *odigosv1.CollectorsGroup, odigosNamespace string, manifestProcessorNames []string, additionalTraceExporters []string, tracesEnabledInClusterCollector bool,
	loadBalancingNeeded bool) config.Config {

	exporters, pipelineExporterNames := tracesExporters(nodeCG, odigosNamespace, tracesEnabledInClusterCollector, loadBalancingNeeded)
	pipelineExporterNames = append(pipelineExporterNames, additionalTraceExporters...)

	tracePipelineProcessors := append([]string{
		batchProcessorName,         // always start with batch
		memoryLimiterProcessorName, // memory limiter is temporary, until we migrate all inputs to rtml based memory protection
		nodeNameProcessorName,
		resourceDetectionProcessorName,
	}, manifestProcessorNames...)
	tracePipelineProcessors = append(tracePipelineProcessors, odigosTrafficMetricsProcessorName) // keep traffic metrics last for most accurate tracking

	return config.Config{
		Receivers: staticTracesReceivers,
		Exporters: exporters,
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				odigosTracesPipelineName: {
					Receivers:  []string{OTLPInReceiverName, odigosEbpfReceiverName},
					Processors: tracePipelineProcessors,
					Exporters:  pipelineExporterNames,
				},
			},
		},
	}
}
