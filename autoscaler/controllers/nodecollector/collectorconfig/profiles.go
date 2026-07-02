package collectorconfig

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
)

// ProfilingPipelineConfig builds the node collector profiles domain when profiling is enabled.
//
// userProcessors are the profiles-capable processors derived from Actions selecting the PROFILES
// signal (see config.CrdProcessorToConfig). They run after the built-in enrichment chain so resource
// and k8s attributes are already populated before they act, and before export to the gateway. Their
// configs are defined in the separate "processors" config domain and merged into the final config.
func ProfilingPipelineConfig(odigosNamespace string, profiling *common.ProfilingConfiguration, userProcessors []string) config.Config {
	if !common.ProfilingPipelineActive(profiling) {
		return config.Config{}
	}

	endpoint := k8sconsts.OtlpGrpcDNSEndpoint(k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace, odigosconsts.OTLPPort)
	exp := commonconf.MergeProfilingOtlpExporter(config.GenericMap{
		"endpoint":    endpoint,
		"tls":         config.GenericMap{"insecure": true},
		"compression": "none",
	}, profiling.Exporter)

	processors := config.GenericMap{
		commonconf.ProfilingNodeFilterProcessor:         commonconf.ProfilingFilterProcessorConfig(),
		commonconf.ProfilingNodeK8sAttributesProcessor:  commonconf.K8sAttributesProfilesProcessorConfig(),
		commonconf.ProfilingNodeOdigosProfilesProcessor: commonconf.OdigosProfilesProcessorConfig(),
		commonconf.ProfilingNodeServiceNameProcessor:    commonconf.ProfilingServiceNameTransformConfig(),
	}
	pipelineProcessors := []string{
		commonconf.ProfilingNodeFilterProcessor,
		commonconf.ProfilingNodeK8sAttributesProcessor,
		commonconf.ProfilingNodeOdigosProfilesProcessor,
	}
	// Native symbolization is opt-in (profiling.symbolization.native). When on, the
	// symbolize processor runs after the keep-filter (only retained profiles are
	// symbolized) and before service-name enrichment.
	if profiling.NativeSymbolizationEnabled() {
		processors[commonconf.ProfilingNodeSymbolizeProcessor] = commonconf.OdigosSymbolizeProcessorConfig()
		pipelineProcessors = append(pipelineProcessors, commonconf.ProfilingNodeSymbolizeProcessor)
	}
	pipelineProcessors = append(pipelineProcessors, commonconf.ProfilingNodeServiceNameProcessor)
	// User processors (from Actions selecting the PROFILES signal) run after the built-in
	// enrichment chain and before export to the gateway.
	pipelineProcessors = append(pipelineProcessors, userProcessors...)

	return config.Config{
		Receivers: config.GenericMap{
			commonconf.ProfilingReceiver: config.GenericMap{},
		},
		Processors: processors,
		Exporters: config.GenericMap{
			commonconf.ProfilingNodeToGatewayExporter: exp,
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				"profiles": {
					Receivers:  []string{commonconf.ProfilingReceiver},
					Processors: pipelineProcessors,
					Exporters:  []string{commonconf.ProfilingNodeToGatewayExporter},
				},
			},
		},
	}
}
