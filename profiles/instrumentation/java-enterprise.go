package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var JavaEnterpriseProfile = profile.Profile{
	ProfileName:      common.ProfileName("java-enterprise"),
	MinimumTier:      common.OnPremOdigosTier,
	ShortDescription: "Instrument Java applications using the java-enterprise distro",
}
