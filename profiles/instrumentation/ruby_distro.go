package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

// LegacyRubyProfile selects the ruby-community-legacy distribution for Ruby 2.7 workloads.
// The distribution and its agent ship only in the enterprise extended odiglet image
// (odiglet.extended=true), so it is opt-in, like LegacyDotNetProfile.
var LegacyRubyProfile = profile.Profile{
	ProfileName:      common.ProfileName("legacy-ruby-instrumentation"),
	MinimumTier:      common.OnPremOdigosTier,
	ShortDescription: "Instrument Ruby 2.7 applications using the legacy OpenTelemetry Ruby distribution (requires the extended odiglet image)",
}
