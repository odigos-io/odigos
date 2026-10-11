package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var EbpfLogCaptureProfile = profile.Profile{
	ProfileName:      common.ProfileName("ebpf-log-capture"),
	MinimumTier:      common.OnPremOdigosTier,
	ShortDescription: "Capture stdout/stderr via eBPF for all languages, correlating logs with active spans where supported (Go, Java)",
}
