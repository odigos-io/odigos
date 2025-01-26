package aggregators

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var GratewallProfile = profile.Profile{
	ProfileName: common.ProfileName("gratewall"),
	MinimumTier: common.OnPremOdigosTier,
	ShortDescription: "Bundle profile that includes " +
		"java-ebpf-instrumentations",
	Dependencies: []common.ProfileName{
		"java-ebpf-instrumentations",
	},
}
