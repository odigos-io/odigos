package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var DisableNodejsExpressProfile = profile.Profile{
	ProfileName:      common.ProfileName("disable-nodejs-express"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Disable express instrumentation in node.js",
}

var NodejsVerbosityFullProfile = profile.Profile{
	ProfileName:      common.ProfileName("nodejs-verbosity-full"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Enable full verbosity for nodejs applications, inclulding all instrumentation libraries",
}
