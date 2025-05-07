package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var ReduceSpanNameCardinalityProfile = profile.Profile{
	ProfileName:      common.ProfileName("reduce-span-name-cardinality"),
	MinimumTier:      common.OnPremOdigosTier,
	ShortDescription: "Reduce the cardinality of the span name by replacing common scenarios observed",
}
