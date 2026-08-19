package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

const (
	odigosTracesLoadbalancingExporterName     = "loadbalancing/traces"
	odigosTracesPipelineName                  = "traces"
	odigosTracesExportingForwardConnectorName = "forward/traces-exporting"
	odigosTracesExportingPipelineName         = "traces/exporting"
	nopExporterName                           = "nop"
)

// tracesPipelineReceivers returns the receiver names wired into the traces pipeline.
func tracesPipelineReceivers(tier common.OdigosTier) []string {
	receivers := []string{OTLPInReceiverName}

	// odigosebpfreceiver only exists in the enterprise collector image - see the comment on
	// odigosEbpfReceiverName in common.go.
	if tier.IsEnterprise() {
		receivers = append(receivers, odigosEbpfReceiverName)
	}

	return receivers
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

			if nodeCG.Spec.OtlpExporterConfiguration != nil && nodeCG.Spec.OtlpExporterConfiguration.Timeout != "" {
				otlpConfig["timeout"] = nodeCG.Spec.OtlpExporterConfiguration.Timeout
			}

			// Add retry_on_failure configuration if present
			if nodeCG.Spec.OtlpExporterConfiguration != nil && nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure != nil {
				otlpConfig["retry_on_failure"] = config.RetryOnFailureConfig(nodeCG.Spec.OtlpExporterConfiguration.RetryOnFailure)
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

type TracesConfigOptions struct {
	CommonSignalConfig
	PostSpanMetricsProcessorNames   []string
	AdditionalTraceExporters        []string
	TracesEnabledInClusterCollector bool
	LoadBalancingNeeded             bool
}

func TracesConfig(nodeCG *odigosv1.CollectorsGroup, opts TracesConfigOptions) config.Config {

	exporters, traceExporterNames := tracesExporters(nodeCG, opts.OdigosNamespace, opts.TracesEnabledInClusterCollector, opts.LoadBalancingNeeded)

	// traces pipeline also feeds the spanmetrics connector.
	// users may want some custom processors (manifestProcessorNames)

	baseProcessors := []string{
		batchProcessorName,         // always start with batch
		memoryLimiterProcessorName, // memory limiter is temporary, until we migrate all inputs to rtml based memory protection
		nodeNameProcessorName,
	}
	if opts.ResourceDetectionEnabled {
		baseProcessors = append(baseProcessors, resourceDetectionProcessorName)
	}
	tracePipelineProcessors := append(baseProcessors, opts.ManifestProcessorNames...)
	tracePipelineProcessors = append(tracePipelineProcessors, odigosTrafficMetricsProcessorName) // keep traffic metrics last for most accurate tracking

	// conditionally, create another pipeline for span exporting,
	// which will run after spanmetrics, but before exporting.
	// the use case is: filter spans in node-collector, but include them in the span metrics.
	connectors := config.GenericMap{}
	tracesMainPipelineExporterNames := []string{}
	additionalPipeline := map[string]config.Pipeline{}
	if len(opts.PostSpanMetricsProcessorNames) == 0 {
		tracesMainPipelineExporterNames = append(traceExporterNames, opts.AdditionalTraceExporters...)
	} else {
		// if we do not have any traces destinations (traceExporterNames == []) but span metrics is enabled, we add a no-op exporter
		if len(traceExporterNames) == 0 {
			exporters[nopExporterName] = config.GenericMap{}
			traceExporterNames = []string{nopExporterName}
		}
		connectors[odigosTracesExportingForwardConnectorName] = config.GenericMap{}
		additionalPipeline[odigosTracesExportingPipelineName] = config.Pipeline{
			Receivers:  []string{odigosTracesExportingForwardConnectorName},
			Processors: opts.PostSpanMetricsProcessorNames,
			Exporters:  traceExporterNames,
		}
		tracesMainPipelineExporterNames = append(opts.AdditionalTraceExporters, odigosTracesExportingForwardConnectorName)
	}

	tracePipeline := map[string]config.Pipeline{
		odigosTracesPipelineName: {
			Receivers:  tracesPipelineReceivers(opts.Tier),
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
