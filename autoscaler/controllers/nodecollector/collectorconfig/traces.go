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
		compression := "none"
		dataCompressionEnabled := nodeCG.Spec.EnableDataCompression
		if dataCompressionEnabled != nil && *dataCompressionEnabled {
			compression = "gzip"
		}

		// Add loadbalancing exporter for traces to ensure consistent gateway routing.
		// This needed for the service graph to work correctly and for the sampling actions to work correctly.
		// If load balancing is not needed, we use the common cluster collector exporter without load balancing.
		if loadBalancingNeeded {
			service := fmt.Sprintf("%s.%s", k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace)
			exporters[odigosTracesLoadbalancingExporterName] = config.GenericMap{
				"protocol": config.GenericMap{
					"otlp": config.GenericMap{
						"compression": compression,
						"tls": config.GenericMap{
							"insecure": true,
						},
					},
				},
				"resolver": config.GenericMap{
					"k8s": config.GenericMap{
						"service": service,
					},
				},
			}
			exporterNames = append(exporterNames, odigosTracesLoadbalancingExporterName)
		} else {
			// Use the common cluster collector exporter, but add compression if needed
			// Note: The actual exporter merge by commonExporters before this function is called.
			// Here we just add it to the exporter name
			exporterNames = append(exporterNames, clusterCollectorExporterName)
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
