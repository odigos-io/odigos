package clustercollector

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
)

// effectiveInsightsConfig returns the insights configuration the gateway should be
// wired for. odigos-insights is an enterprise-only component - helm renders its
// Deployment, Services and ClickHouse only when an on-prem token is available - so on
// community tier the flag is ignored. Otherwise the gateway would force the traces
// signal on cluster-wide, install groupbytrace in front of it and export every span to
// a Service that was never deployed. Same reasoning as the tier gate around
// addProfilingGatewayPipeline in syncConfigMap.
func effectiveInsightsConfig(insights *common.InsightsConfiguration, tier common.OdigosTier) *common.InsightsConfiguration {
	if !tier.IsEnterprise() {
		return nil
	}
	return insights
}

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

	// Target the headless insights Service via the dns:/// resolver and enable
	// round_robin so the gateway's gRPC client fans out across every insights
	// pod instead of pinning one behind the ClusterIP VIP.
	c.Exporters[commonconf.InsightsGatewayExporter] = config.GenericMap{
		"endpoint":      k8sconsts.InsightsOtlpGrpcDNSEndpoint(odigosNs),
		"balancer_name": "round_robin",
		"tls":           config.GenericMap{"insecure": true},
		"compression":   "none",
		"retry_on_failure": config.GenericMap{
			"enabled": false,
		},
	}

	rootPipeline.Exporters = append(rootPipeline.Exporters, commonconf.InsightsGatewayExporter)
	c.Service.Pipelines[rootPipelineName] = rootPipeline

	return nil
}
