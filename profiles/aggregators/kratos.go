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
		"category-attributes",
		"copy-scope",
		"hostname-as-podname",
		"code-attributes",
		"query-operation-detector",
		"small-batches",
		"size_m",
		"allow_concurrent_agents",
		"mount-method-k8s-host-path",
		"reduce-span-name-cardinality",
		"disable-gin",
	},
	ModifyConfigFunc: func(config *common.OdigosConfiguration) {
		rollbackDisabled := true
		config.RollbackDisabled = &rollbackDisabled
	},
}
