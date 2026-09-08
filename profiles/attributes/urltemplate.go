package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var UrlTemplateProfile = profile.Profile{
	ProfileName:      common.ProfileName("url-template"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Replace dynamic URL path segments with templates so HTTP span names and attributes stay low-cardinality",
}
