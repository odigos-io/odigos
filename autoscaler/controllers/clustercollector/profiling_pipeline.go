package clustercollector

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
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

	endpoint := k8sconsts.UiOtlpGrpcEndpoint(odigosNs)

	exp := commonconf.MergeProfilingOtlpExporter(config.GenericMap{
		"endpoint":    endpoint,
		"tls":         config.GenericMap{"insecure": true},
		"compression": "none",
	}, profiling.Exporter)

	c.Exporters[commonconf.ProfilingGatewayToUIExporter] = exp

	c.Service.Pipelines["profiles"] = config.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: nil,
		Exporters:  []string{commonconf.ProfilingGatewayToUIExporter},
	}
	return nil
}
