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

func tracesExporters(nodeCG *odigosv1.CollectorsGroup, odigosNamespace string) config.GenericMap {
	compression := "none"
	dataCompressionEnabled := nodeCG.Spec.EnableDataCompression
	if dataCompressionEnabled != nil && *dataCompressionEnabled {
		compression = "gzip"
	}

	service := fmt.Sprintf("%s.%s", k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace)

	// Add loadbalancing exporter for traces to ensure consistent gateway routing.
	// This allows the servicegraph connector to properly aggregate trace data
	// by sending all traces from a node collector to the same gateway instance.
	return config.GenericMap{
		odigosTracesLoadbalancingExporterName: config.GenericMap{
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
		},
	}
}

func TracesConfig(nodeCG *odigosv1.CollectorsGroup, odigosNamespace string, manifestProcessorNames []string) config.Config {

	tracePipelineProcessors := append([]string{
		BatchProcessorName,         // always start with batch
		MemoryLimiterProcessorName, // memory limiter is temporary, until we migrate all inputs to rtml based memory protection
		NodeNameProcessorName,
		ResourceDetectionProcessorName,
	}, manifestProcessorNames...)
	tracePipelineProcessors = append(tracePipelineProcessors, OdigosTrafficMetricsProcessorName) // keep traffic metrics last for most accurate tracking

	return config.Config{
		Receivers: staticTracesReceivers,
		Exporters: tracesExporters(nodeCG, odigosNamespace),
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				odigosTracesPipelineName: {
					Receivers:  []string{OTLPInReceiverName, odigosEbpfReceiverName},
					Processors: tracePipelineProcessors,
					Exporters:  []string{odigosTracesLoadbalancingExporterName},
				},
			},
		},
	}
}
