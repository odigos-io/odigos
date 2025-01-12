package pipeline

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var DisableNameProcessorProfile = profile.Profile{
	ProfileName:      common.ProfileName("disable-name-processor"),
	MinimumTier:      common.OdigosTier(common.CommunityOdigosTier),
	ShortDescription: "If not using dotnet or java native instrumentations, disable the name processor which is not needed",
}
