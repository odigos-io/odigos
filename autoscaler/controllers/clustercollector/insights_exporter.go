package clustercollector

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
)

// addInsightsGatewayExporter appends an OTLP gRPC exporter to the gateway's
// root traces pipeline so every processed span fans out to the in-cluster
// sidecar alongside the destination router. Noop when disabled or when no
// root traces pipeline exists.
func addInsightsGatewayExporter(c *config.Config, odigosNs string, insights *common.InsightsConfiguration) error {
	if !common.InsightsPipelineActive(insights) {
		return nil
	}

	rootPipelineName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	rootPipeline, hasRoot := c.Service.Pipelines[rootPipelineName]
	if !hasRoot {
		return nil
	}

	if c.Exporters == nil {
		c.Exporters = config.GenericMap{}
	}

	c.Exporters[commonconf.InsightsGatewayExporter] = config.GenericMap{
		"endpoint":    k8sconsts.InsightsOtlpGrpcEndpoint(odigosNs),
		"tls":         config.GenericMap{"insecure": true},
		"compression": "none",
		"retry_on_failure": config.GenericMap{
			"enabled": false,
		},
	}

	rootPipeline.Exporters = append(rootPipeline.Exporters, commonconf.InsightsGatewayExporter)
	c.Service.Pipelines[rootPipelineName] = rootPipeline

	return nil
}
