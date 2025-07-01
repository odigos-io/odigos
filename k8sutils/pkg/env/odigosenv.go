package env

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
)

// return if LD_PRELOAD is set in the envs list, and it's value if it does
func FindLdPreloadInEnvs(envs []odigosv1.EnvVar) (string, bool) {
	// list is expected to contain 0-2 elements, so we can use a simple loop.
	for i := range envs {
		if envs[i].Name == consts.LdPreloadEnvVarName {
			return envs[i].Value, true
		}
	}
	return "", false
}
