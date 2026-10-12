package clustercollector

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/odigos-io/odigos/instrumentor/controllers/pipeline"
)

const gatewayProfilesPipeline = "profiles"

// addInterrogationExporters enables the interrogation exporters when interrogation
// is on. Profiles exporter is appended only when the gateway profiles pipeline
// exists. Traces exporter is appended to the root traces pipeline (traces/in),
// which is post-groupbytrace — the same attachment point as insights and
// service I/O correlations. Profile/trace correlation happens in ClickHouse.
// transactionIdentity supplies optional dimensions for the traces exporter config
// (from OdigosConfiguration.transactionIdentity).
// groupByTraceWaitDuration is the gateway groupbytrace wait_duration (e.g. "30s");
// the profiles exporter uses 2× that as the Redis whole-trace key TTL.
func addInterrogationExporters(c *config.Config, odigosNs string, interrogation *common.InterrogationConfiguration, transactionIdentity *common.TransactionIdentityConfiguration, groupByTraceWaitDuration string) error {
	if !common.InterrogationActive(interrogation) {
		return nil
	}

	profilesPipeline, hasProfiles := c.Service.Pipelines[gatewayProfilesPipeline]
	rootPipelineName := pipelinegen.GetTelemetryRootPipelineName(common.TracesObservabilitySignal)
	rootPipeline, hasTraces := c.Service.Pipelines[rootPipelineName]
	if !hasProfiles && !hasTraces {
		return nil
	}

	if c.Exporters == nil {
		c.Exporters = config.GenericMap{}
	}

	if hasProfiles {
		profilesExp := config.GenericMap{
			"clickhouse_endpoint": k8sconsts.InsightsClickHouseEndpoint(odigosNs),
			"clickhouse_password": "${" + k8sconsts.OdigosInsightsClickHousePasswordEnv + "}",
			"redis_endpoint":      k8sconsts.OdigosCacheEndpoint(odigosNs),
		}
		if groupByTraceWaitDuration != "" {
			profilesExp["group_by_trace_wait_duration"] = groupByTraceWaitDuration
		}
		c.Exporters[pipeline.InterrogationProfilesExporter] = profilesExp
		profilesPipeline.Exporters = append(profilesPipeline.Exporters, pipeline.InterrogationProfilesExporter)
		c.Service.Pipelines[gatewayProfilesPipeline] = profilesPipeline
	}

	if hasTraces {
		tracesExp := config.GenericMap{
			"clickhouse_endpoint": k8sconsts.InsightsClickHouseEndpoint(odigosNs),
			"clickhouse_password": "${" + k8sconsts.OdigosInsightsClickHousePasswordEnv + "}",
			"redis_endpoint":      k8sconsts.OdigosCacheEndpoint(odigosNs),
		}
		if dims := transactionIdentityDimensions(transactionIdentity); len(dims) > 0 {
			tracesExp["transaction_identity_dimensions"] = dims
		}
		c.Exporters[pipeline.InterrogationTracesExporter] = tracesExp
		rootPipeline.Exporters = append(rootPipeline.Exporters, pipeline.InterrogationTracesExporter)
		c.Service.Pipelines[rootPipelineName] = rootPipeline
	}

	return nil
}

func transactionIdentityDimensions(cfg *common.TransactionIdentityConfiguration) []string {
	if cfg == nil || len(cfg.Dimensions) == 0 {
		return nil
	}
	return cfg.Dimensions
}
