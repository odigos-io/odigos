package clustercollector

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/odigos-io/odigos/instrumentor/controllers/pipeline"
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
	c.Exporters[pipeline.InsightsGatewayExporter] = config.GenericMap{
		"endpoint":      k8sconsts.InsightsOtlpGrpcDNSEndpoint(odigosNs),
		"balancer_name": "round_robin",
		"tls":           config.GenericMap{"insecure": true},
		"compression":   "none",
		"retry_on_failure": config.GenericMap{
			"enabled": false,
		},
		// groupbytrace releases each trace as its own ConsumeTraces call, so
		// without batching every trace becomes a separate Export RPC. max_size
		// is left unset (0) on purpose: a positive value splits requests by span
		// count and can cut a trace across two Exports, which insights needs whole.
		// https://github.com/open-telemetry/opentelemetry-collector/blob/v0.159.0/exporter/exporterhelper/internal/queuebatch/traces_batch.go#L37-L40
		"sending_queue": config.GenericMap{
			"batch": config.GenericMap{
				"sizer":         "items",
				"min_size":      1000,
				"flush_timeout": "200ms",
			},
		},
	}

	rootPipeline.Exporters = append(rootPipeline.Exporters, pipeline.InsightsGatewayExporter)
	c.Service.Pipelines[rootPipelineName] = rootPipeline

	return nil
}
