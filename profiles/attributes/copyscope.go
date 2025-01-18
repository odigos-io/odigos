package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var CopyScopeProfile = profile.Profile{
	ProfileName:      common.ProfileName("copy-scope"),
	MinimumTier:      common.OdigosTier(common.CommunityOdigosTier),
	ShortDescription: "Copy the scope name into a separate attribute for backends that do not support scopes",
}
