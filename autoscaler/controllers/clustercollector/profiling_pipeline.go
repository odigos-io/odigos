package clustercollector

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
)

func addProfilingGatewayPipeline(c *config.Config, odigosNs string, profiling *common.ProfilingConfiguration, gateway *odigosv1.CollectorsGroup) error {
	if !common.ProfilingPipelineActive(profiling) {
		return nil
	}
	var rs odigosv1.CollectorsGroupResourcesSettings
	if gateway != nil {
		rs = gateway.Spec.ResourcesSettings
	}
	// Filter + k8s_attributes run on the node collector only; OTLP to the gateway already carries
	// enriched resource attributes. The gateway profiles pipeline is receive → export to UI.
	if c.Exporters == nil {
		c.Exporters = config.GenericMap{}
	}
	if c.Service.Pipelines == nil {
		c.Service.Pipelines = map[string]config.Pipeline{}
	}

	endpoint := k8sconsts.UiOtlpGrpcEndpoint(odigosNs)

	exp := commonconf.MergeProfilingOtlpExporter(config.GenericMap{
		"endpoint":    endpoint,
		"tls":         config.GenericMap{"insecure": true},
		"compression": "none",
	}, profiling.Exporter)

	c.Exporters[commonconf.ProfilingGatewayToUIExporter] = exp

	if c.Processors == nil {
		c.Processors = config.GenericMap{}
	}
	if _, ok := c.Processors["memory_limiter"]; !ok {
		c.Processors["memory_limiter"] = commonconf.GetMemoryLimiterConfig(rs)
	}
	if _, ok := c.Processors[odigosconsts.GenericBatchProcessorConfigKey]; !ok {
		c.Processors[odigosconsts.GenericBatchProcessorConfigKey] = config.GenericMap{}
	}

	c.Service.Pipelines["profiles"] = config.Pipeline{
		Receivers: []string{"otlp"},
		Processors: []string{
			"memory_limiter",
			odigosconsts.GenericBatchProcessorConfigKey,
		},
		Exporters: []string{commonconf.ProfilingGatewayToUIExporter},
	}
	return nil
}
