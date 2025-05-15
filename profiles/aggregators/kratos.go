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
	ManifestNames: []string{
		"kratos-attr-replicaset.yaml",
		"kratos-hostname-as-podname.yaml",
		"kratos-reduce-span-name-cardinality.yaml",
		"kratos-category-attributes.yaml",
	},
	Dependencies: []common.ProfileName{
		"db-payload-collection",
		"semconv",
		"copy-scope",
		"code-attributes",
		"query-operation-detector",
		"small-batches",
		"size_m",
		"allow_concurrent_agents",
		"mount-method-k8s-host-path",
		"avoid-java-opts-env-var",
	},
}
