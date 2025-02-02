package aggregators

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var GreatwallProfile = profile.Profile{
	ProfileName: common.ProfileName("greatwall"),
	MinimumTier: common.OnPremOdigosTier,
	ShortDescription: "Bundle profile that includes " +
		"java-ebpf-instrumentations and legacy-dotnet-instrumentation",
	Dependencies: []common.ProfileName{
		"java-ebpf-instrumentations",
		"legacy-dotnet-instrumentation",
	},
}
