package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var PythonEbpfProfile = profile.Profile{
	ProfileName:      common.ProfileName("python-ebpf"),
	MinimumTier:      common.OnPremOdigosTier,
	ShortDescription: "Instrument Python applications out of process with the python-ebpf distro (no changes to the application)",
}
