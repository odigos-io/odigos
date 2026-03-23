package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var JavaEbpfExtensionProfile = profile.Profile{
	ProfileName:      common.ProfileName("java-ebpf-extension"),
	MinimumTier:      common.OnPremOdigosTier,
	ShortDescription: "Instrument Java applications using eBPF extension (otel_agent_extension)",
}
