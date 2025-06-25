package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var DisableGinProfile = profile.Profile{
	ProfileName:      common.ProfileName("disable-gin"),
	MinimumTier:      common.OnPremOdigosTier,
	ShortDescription: "Disable gin-gonic/gin instrumentation",
}
