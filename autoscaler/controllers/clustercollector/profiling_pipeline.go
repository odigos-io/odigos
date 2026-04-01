package clustercollector

import (
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
)

func addProfilingGatewayPipeline(c *config.Config, odigosNs string, profiling *common.ProfilingConfiguration) error {
	if !common.ProfilingPipelineActive(profiling) {
		return nil
	}
	// Filter + k8s_attributes run on the node collector only; OTLP to the gateway already carries
	// enriched resource attributes. The gateway profiles pipeline is receive → export to UI.
	if c.Exporters == nil {
		c.Exporters = config.GenericMap{}
	}
	if c.Service.Pipelines == nil {
		c.Service.Pipelines = map[string]config.Pipeline{}
	}

	endpoint := odigosconsts.UiOtlpGrpcEndpoint(odigosNs)

	exp := commonconf.MergeProfilingOtlpExporter(config.GenericMap{
		"endpoint":    endpoint,
		"tls":         config.GenericMap{"insecure": true},
		"compression": "none",
	}, profiling.Exporter)

	c.Exporters[odigosconsts.ProfilingGatewayToUIExporter] = exp

	c.Service.Pipelines["profiles"] = config.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: nil,
		Exporters:  []string{odigosconsts.ProfilingGatewayToUIExporter},
	}
	return nil
}
