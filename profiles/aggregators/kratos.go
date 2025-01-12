package aggregators

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var KratosProfile = profile.Profile{
	ProfileName:      common.ProfileName("kratos"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Bundle profile that includes db-payload-collection, semconv, category-attributes, copy-scope, hostname-as-podname, code-attributes, query-operation-detector, disableNameProcessorProfile, small-batches, size_m, allow_concurrent_agents",
	Dependencies:     []common.ProfileName{"db-payload-collection", "semconv", "category-attributes", "copy-scope", "hostname-as-podname", "code-attributes", "query-operation-detector", "disable-name-processor", "small-batches", "size_m", "allow_concurrent_agents"},
}
