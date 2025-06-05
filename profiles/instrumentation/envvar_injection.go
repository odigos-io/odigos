package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var LoaderFallbackToPodManifestEnvVarInjection = profile.Profile{
	ProfileName:      common.ProfileName("loader-fallback-to-pod-manifest-env-var-injection"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Try using odigos loader env var injection method, fallback to pod manifest if not possible",
	ModifyConfigFunc: func(config *common.OdigosConfiguration) {
		method := common.LoaderFallbackToPodManifestInjectionMethod
		if config.AgentEnvVarsInjectionMethod == nil {
			config.AgentEnvVarsInjectionMethod = &method
		}
	},
}
