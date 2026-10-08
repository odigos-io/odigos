package collectorconfig

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/instrumentor/controllers/pipeline"
)

// ProfilingPipelineConfig builds the node collector profiles domain when profiling is enabled.
func ProfilingPipelineConfig(odigosNamespace string, profiling *common.ProfilingConfiguration, manifestProcessorNames []string) config.Config {
	if !common.ProfilingPipelineActive(profiling) {
		return config.Config{}
	}

	endpoint := k8sconsts.OtlpGrpcDNSEndpoint(k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace, odigosconsts.OTLPPort)
	exp := pipeline.MergeProfilingOtlpExporter(config.GenericMap{
		"endpoint":    endpoint,
		"tls":         config.GenericMap{"insecure": true},
		"compression": "none",
	}, profiling.Exporter)

	// memory_limiter itself is defined once, globally, by commonProcessors() (see
	// common.go) — every pipeline just references its name, never redefines it.
	processors := config.GenericMap{
		pipeline.ProfilingNodeFilterProcessor:         pipeline.ProfilingFilterProcessorConfig(),
		pipeline.ProfilingNodeK8sAttributesProcessor:  pipeline.K8sAttributesProfilesProcessorConfig(),
		pipeline.ProfilingNodeOdigosProfilesProcessor: pipeline.OdigosProfilesProcessorConfig(),
		pipeline.ProfilingNodeServiceNameProcessor:    pipeline.ProfilingServiceNameTransformConfig(),
	}
	pipelineProcessors := []string{
		memoryLimiterProcessorName,
		pipeline.ProfilingNodeFilterProcessor,
		pipeline.ProfilingNodeK8sAttributesProcessor,
		pipeline.ProfilingNodeOdigosProfilesProcessor,
	}
	// Native symbolization is opt-in (profiling.symbolization.native). When on, the
	// symbolize processor runs after the keep-filter (only retained profiles are
	// symbolized) and before service-name enrichment.
	if profiling.NativeSymbolizationEnabled() {
		processors[pipeline.ProfilingNodeSymbolizeProcessor] = pipeline.OdigosSymbolizeProcessorConfig()
		pipelineProcessors = append(pipelineProcessors, pipeline.ProfilingNodeSymbolizeProcessor)
	}
	pipelineProcessors = append(pipelineProcessors, pipeline.ProfilingNodeServiceNameProcessor)
	pipelineProcessors = append(pipelineProcessors, manifestProcessorNames...)
	pipelineProcessors = append(pipelineProcessors, odigosTrafficMetricsProcessorName) // keep traffic metrics last for most accurate tracking

	return config.Config{
		Receivers: config.GenericMap{
			pipeline.ProfilingReceiver: config.GenericMap{},
		},
		Processors: processors,
		Exporters: config.GenericMap{
			pipeline.ProfilingNodeToGatewayExporter: exp,
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				"profiles": {
					Receivers:  []string{pipeline.ProfilingReceiver},
					Processors: pipelineProcessors,
					Exporters:  []string{pipeline.ProfilingNodeToGatewayExporter},
				},
			},
		},
	}
}
