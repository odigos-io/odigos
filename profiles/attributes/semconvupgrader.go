package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var SemconvUpgraderProfile = profile.Profile{
	ProfileName:      common.ProfileName("semconv"),
	MinimumTier:      common.OdigosTier(common.CommunityOdigosTier),
	ShortDescription: "Upgrade and align some attribute names to a newer version of the OpenTelemetry semantic conventions",
}
