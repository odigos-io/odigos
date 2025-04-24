package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var AvoidInjectingJavaOptsEnvVar = profile.Profile{
	ProfileName:      common.ProfileName("avoid-java-opts-env-var"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Avoid injecting the Odigos value in JAVA_OPTS environment variable into Java applications",
	ModifyConfigFunc: func(config *common.OdigosConfiguration) {
		config.AvoidInjectingJavaOptsEnvVar = true
	},
}