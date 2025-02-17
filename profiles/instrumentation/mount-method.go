package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var MountMethodK8sHostPathProfile = profile.Profile{
	ProfileName:      common.ProfileName("mount-method-k8s-host-path"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Mount odigos agents files to pod container filesystem using k8s host path volume",
	ModifyConfigFunc: func(config *common.OdigosConfiguration) {
		// avoid overriding user's choice if it was already set
		// TODO: this implies semantics of first setter wins, which is not ideal
		// and not consistent with yaml profiles (where the later one overrides)
		if config.MountMethod == nil {
			hostPath := common.K8sHostPathMountMethod
			config.MountMethod = &hostPath
		}
	},
}

var MountMethodK8sVirtualDevice = profile.Profile{
	ProfileName:      common.ProfileName("mount-method-k8s-virtual-device"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Mount odigos agents files to pod container filesystem using k8s virtual device.",
	ModifyConfigFunc: func(config *common.OdigosConfiguration) {
		virtualDevice := common.K8sVirtualDeviceMountMethod
		config.MountMethod = &virtualDevice
	},
}
