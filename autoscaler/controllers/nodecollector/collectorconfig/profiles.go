package collectorconfig

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
)

// ProfilingPipelineConfig builds the node collector profiles domain when profiling is enabled.
func ProfilingPipelineConfig(odigosNamespace string, profiling *common.ProfilingConfiguration) config.Config {
	if !common.ProfilingPipelineActive(profiling) {
		return config.Config{}
	}

	endpoint := odigosconsts.OtlpGrpcDNSEndpoint(k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace, odigosconsts.OTLPPort)
	exp := commonconf.MergeProfilingOtlpExporter(config.GenericMap{
		"endpoint":    endpoint,
		"tls":         config.GenericMap{"insecure": true},
		"compression": "none",
	}, profiling.Exporter)

	return config.Config{
		Receivers: config.GenericMap{
			odigosconsts.ProfilingReceiver: config.GenericMap{},
		},
		Processors: config.GenericMap{
			odigosconsts.ProfilingNodeFilterProcessor:        commonconf.ProfilingFilterProcessorConfig(),
			odigosconsts.ProfilingNodeK8sAttributesProcessor: commonconf.K8sAttributesProfilesProcessorConfig(),
		},
		Exporters: config.GenericMap{
			odigosconsts.ProfilingNodeToGatewayExporter: exp,
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				"profiles": {
					Receivers:  []string{odigosconsts.ProfilingReceiver},
					Processors: []string{odigosconsts.ProfilingNodeFilterProcessor, odigosconsts.ProfilingNodeK8sAttributesProcessor},
					Exporters:  []string{odigosconsts.ProfilingNodeToGatewayExporter},
				},
			},
		},
	}
}
