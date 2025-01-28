package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var LegacyDotNetProfile = profile.Profile{
	ProfileName:      common.ProfileName("legacy-dotnet-instrumentation"),
	MinimumTier:      common.OnPremOdigosTier,
	ShortDescription: "Instrument DotNet applications using legacy OpenTelemetry instrumentation (needed for 6.0 support)",
}
