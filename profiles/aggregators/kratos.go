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
		config.MetricsSources.AgentMetrics.RuntimeMetrics.Java.Disabled = &disabled
	},
}
