package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var LabelAttributeProfile = profile.Profile{
	ProfileName:      common.ProfileName("label-attributes"),
	MinimumTier:      common.OnPremOdigosTier,
	ShortDescription: "Add pod's labels attributes to the spans",
}
