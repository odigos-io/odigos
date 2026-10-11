package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var PodManifestEnvVarInjection = profile.Profile{
	ProfileName:      common.ProfileName("pod-manifest-env-var-injection"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Use the pod manifest to add odigos runtime specific environment variables",
	ModifyConfigFunc: func(config *common.OdigosConfiguration) {
		method := common.PodManifestEnvInjectionMethod
		if config.AgentEnvVarsInjectionMethod == nil {
			config.AgentEnvVarsInjectionMethod = &method
		}
	},
}
