package aggregators

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var GreatwallProfile = profile.Profile{
	ProfileName: common.ProfileName("greatwall"),
	MinimumTier: common.OnPremOdigosTier,
	ShortDescription: "Bundle profile that includes " +
		"specific preset for on-premises installations.",
	Dependencies: []common.ProfileName{
		"java-ebpf-instrumentations",
		"legacy-dotnet-instrumentation",
		"mount-method-k8s-virtual-device",
	},
	ModifyConfigFunc: func(config *common.OdigosConfiguration) {
		// temporary set in profile until we add auto discovery for /var/log symlink target
		if config.CollectorNode == nil {
			config.CollectorNode = &common.CollectorNodeConfiguration{}
		}
		if config.CollectorNode.K8sNodeLogsDirectory == "" {
			config.CollectorNode.K8sNodeLogsDirectory = "/mnt/var/log"
		}
	},
}
