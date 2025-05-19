package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

// Used as a migration path - eventually this will be the default
// and we will never add to the JAVA_OPTS env var
// This profile is used to avoid injecting the Odigos value in JAVA_OPTS environment variable into Java applications
var AvoidInjectingJavaOptsEnvVar = profile.Profile{
	ProfileName:      common.ProfileName("avoid-java-opts-env-var"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Avoid injecting the Odigos value in JAVA_OPTS environment variable into Java applications",
	ModifyConfigFunc: func(config *common.OdigosConfiguration) {
		b := true
		if config.AvoidInjectingJavaOptsEnvVar == nil {
			config.AvoidInjectingJavaOptsEnvVar = &b
		}
	},
}

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
