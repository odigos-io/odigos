package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

const (
	odigosTracesLoadbalancingExporterName     = "loadbalancing/traces"
	odigosTracesPipelineName                  = "traces"
	odigosTracesExportingForwardConnectorName = "forward/traces-exporting"
	odigosTracesExportingPipelineName         = "traces/exporting"
)

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

			if nodeCG.Spec.OtlpExporterConfiguration != nil && nodeCG.Spec.OtlpExporterConfiguration.Timeout != "" {
				otlpConfig["timeout"] = nodeCG.Spec.OtlpExporterConfiguration.Timeout
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
			exporterNames = append(exporterNames, clusterCollectorTracesExporterName)
		}
	}

	return exporters, exporterNames
}

func TracesConfig(nodeCG *odigosv1.CollectorsGroup, odigosNamespace string, manifestProcessorNames []string, postSpanMetricsProcessorNames []string, additionalTraceExporters []string, tracesEnabledInClusterCollector bool,
	loadBalancingNeeded bool) config.Config {

	exporters, traceExporterNames := tracesExporters(nodeCG, odigosNamespace, tracesEnabledInClusterCollector, loadBalancingNeeded)

	// traces pipeline also feeds the spanmetrics connector.
	// users may want some custom processors (manifestProcessorNames)

	tracePipelineProcessors := append([]string{
		batchProcessorName,         // always start with batch
		memoryLimiterProcessorName, // memory limiter is temporary, until we migrate all inputs to rtml based memory protection
		nodeNameProcessorName,
		resourceDetectionProcessorName,
	}, manifestProcessorNames...)
	tracePipelineProcessors = append(tracePipelineProcessors, odigosTrafficMetricsProcessorName) // keep traffic metrics last for most accurate tracking

	// conditionally, create another pipeline for span exporting,
	// which will run after spanmetrics, but before exporting.
	// the use case is: filter spans in node-collector, but include them in the span metrics.
	connectors := config.GenericMap{}
	tracesMainPipelineExporterNames := []string{}
	additionalPipeline := map[string]config.Pipeline{}
	if len(postSpanMetricsProcessorNames) == 0 {
		tracesMainPipelineExporterNames = append(traceExporterNames, additionalTraceExporters...)
	} else {
		connectors[odigosTracesExportingForwardConnectorName] = config.GenericMap{}
		additionalPipeline[odigosTracesExportingPipelineName] = config.Pipeline{
			Receivers:  []string{odigosTracesExportingForwardConnectorName},
			Processors: postSpanMetricsProcessorNames,
			Exporters:  traceExporterNames,
		}
		tracesMainPipelineExporterNames = append(additionalTraceExporters, odigosTracesExportingForwardConnectorName)
	}

	tracePipeline := map[string]config.Pipeline{
		odigosTracesPipelineName: {
			Receivers:  []string{OTLPInReceiverName, odigosEbpfReceiverName},
			Processors: tracePipelineProcessors,
			Exporters:  tracesMainPipelineExporterNames,
		},
	}
	if len(additionalPipeline) > 0 {
		for pipelineName, pipeline := range additionalPipeline {
			tracePipeline[pipelineName] = pipeline
		}
	}

	return config.Config{
		Connectors: connectors,
		Exporters:  exporters,
		Service: config.Service{
			Pipelines: tracePipeline,
		},
	}
}
