package clustercollector

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/instrumentor/controllers/pipeline"
)

const gatewayProfilesPipeline = "profiles"

// addInterrogationExporters enables the interrogation profiles exporter when
// interrogation is on and the gateway profiles pipeline exists. Profile stacks
// land in Redis; insights expands matching stacks into tx_call_trie in ClickHouse.
// groupByTraceWaitDuration is the gateway groupbytrace wait_duration (e.g. "30s");
// the profiles exporter uses 2× that as the Redis whole-trace key TTL.
func addInterrogationExporters(c *config.Config, odigosNs string, interrogation *common.InterrogationConfiguration, groupByTraceWaitDuration string) error {
	if !common.InterrogationActive(interrogation) {
		return nil
	}

	profilesPipeline, hasProfiles := c.Service.Pipelines[gatewayProfilesPipeline]
	if !hasProfiles {
		return nil
	}

	if c.Exporters == nil {
		c.Exporters = config.GenericMap{}
	}

	profilesExp := config.GenericMap{
		"redis_endpoint": k8sconsts.OdigosCacheEndpoint(odigosNs),
	}
	if groupByTraceWaitDuration != "" {
		profilesExp["group_by_trace_wait_duration"] = groupByTraceWaitDuration
	}
	c.Exporters[pipeline.InterrogationProfilesExporter] = profilesExp
	profilesPipeline.Exporters = append(profilesPipeline.Exporters, pipeline.InterrogationProfilesExporter)
	c.Service.Pipelines[gatewayProfilesPipeline] = profilesPipeline

	return nil
}
