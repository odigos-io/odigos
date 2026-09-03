package aggregators

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var InsightsProfile = profile.Profile{
	ProfileName: common.ProfileName("insights"),
	MinimumTier: common.CommunityOdigosTier,
	ShortDescription: "Bundle profile that includes " +
		"specific presets for odigos insights.",
	Dependencies: []common.ProfileName{
		"infer-db-attributes",
		"url-template",
	},
}
