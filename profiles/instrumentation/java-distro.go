package instrumentation

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var JavaNativeInstrumentationsProfile = profile.Profile{
	ProfileName:      common.ProfileName("java-native-instrumentations"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Deprecated, native instrumentations are now enabled by default",
}
var JavaEbpfInstrumentationsProfile = profile.Profile{
	ProfileName:      common.ProfileName("java-ebpf-instrumentations"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Instrument Java applications using eBPF instrumentation and eBPF enterprise processing",
	KubeObject:       &odigosv1alpha1.InstrumentationRule{},
}
