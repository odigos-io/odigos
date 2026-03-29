package collectorconfig

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
)

const (
	profilingReceiverName             = "profiling"
	filterProfilesProcessor           = "filter/profiles-require-container-id"
	k8sAttributesProfilesProcessor    = "k8s_attributes/profiles"
	otlpProfilesToGatewayExporterName = "otlp_grpc/profiles-to-gateway"
)

// ProfilingPipelineConfig builds the node collector profiles domain when profiling is enabled.
func ProfilingPipelineConfig(odigosNamespace string, profiling *common.ProfilingConfiguration) config.Config {
	if profiling == nil || profiling.Enabled == nil || !*profiling.Enabled {
		return config.Config{}
	}

	endpoint := fmt.Sprintf("dns:///%s.%s:%d", k8sconsts.OdigosClusterCollectorServiceName, odigosNamespace, odigosconsts.OTLPPort)
	exp := commonconf.MergeProfilingOtlpExporter(config.GenericMap{
		"endpoint":    endpoint,
		"tls":         config.GenericMap{"insecure": true},
		"compression": "none",
	}, profiling.Exporter)

	return config.Config{
		Receivers: config.GenericMap{
			profilingReceiverName: config.GenericMap{},
		},
		Processors: config.GenericMap{
			filterProfilesProcessor:        commonconf.ProfilingFilterProcessorConfig(),
			k8sAttributesProfilesProcessor: commonconf.K8sAttributesProfilesProcessorConfig(),
		},
		Exporters: config.GenericMap{
			otlpProfilesToGatewayExporterName: exp,
		},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				"profiles": {
					Receivers:  []string{profilingReceiverName},
					Processors: []string{filterProfilesProcessor, k8sAttributesProfilesProcessor},
					Exporters:  []string{otlpProfilesToGatewayExporterName},
				},
			},
		},
	}
}
