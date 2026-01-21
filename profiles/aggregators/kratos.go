package aggregators

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var KratosProfile = profile.Profile{
	ProfileName: common.ProfileName("kratos"),
	MinimumTier: common.OnPremOdigosTier,
	ShortDescription: "Bundle profile that includes " +
		"specific presets for on-premises installations.",
	Dependencies: []common.ProfileName{
		"db-payload-collection",
		"semconv",
		// "category-attributes",
		"copy-scope",
		"hostname-as-podname",
		"code-attributes",
		// "query-operation-detector", - disabled to be run in data collection collector
		// "small-batches", - issue is fixed in receiver. can now handle 85MB
		"allow_concurrent_agents",
		"reduce-span-name-cardinality",
		"disable-gin",
		"semconvdynamo",
		"semconvredis",
	},
	ModifyConfigFunc: func(config *common.OdigosConfiguration) {
		disabled := true
		config.RollbackDisabled = &disabled

		// Ensure each level exists, creating only if nil
		if config.MetricsSources == nil {
			config.MetricsSources = &common.MetricsSourceConfiguration{}
		}
		if config.MetricsSources.AgentMetrics == nil {
			config.MetricsSources.AgentMetrics = &common.MetricsSourceAgentMetricsConfiguration{}
		}
		if config.MetricsSources.AgentMetrics.RuntimeMetrics == nil {
			config.MetricsSources.AgentMetrics.RuntimeMetrics = &common.MetricsSourceAgentRuntimeMetricsConfiguration{}
		}
		if config.MetricsSources.AgentMetrics.RuntimeMetrics.Java == nil {
			config.MetricsSources.AgentMetrics.RuntimeMetrics.Java = &common.MetricsSourceAgentJavaRuntimeMetricsConfiguration{}
		}
		config.MetricsSources.AgentMetrics.RuntimeMetrics.Java.Disabled = &disabled
	},
}
