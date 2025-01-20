package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var CodeAttributesProfile = profile.Profile{
	ProfileName:      common.ProfileName("code-attributes"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Record span attributes in 'code' namespace where supported",
}
