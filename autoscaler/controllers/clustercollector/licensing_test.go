package clustercollector

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestCollectorLicenseEnvironment(t *testing.T) {
	t.Setenv("ODIGOS_LICENSE_PROVIDER", "")
	env := collectorLicenseEnv(common.OnPremOdigosTier)
	require.Len(t, env, 1)
	require.Equal(t, "odigos-onprem-token", env[0].ValueFrom.SecretKeyRef.Key)

	t.Setenv("ODIGOS_LICENSE_PROVIDER", "aws-marketplace")
	env = collectorLicenseEnv(common.OnPremOdigosTier)
	require.Len(t, env, 1)
	require.Equal(t, "ODIGOS_LICENSE_PROVIDER", env[0].Name)
	require.Equal(t, "aws-marketplace", env[0].Value)
	require.Nil(t, env[0].ValueFrom)
	require.Nil(t, collectorLicenseEnv(common.CommunityOdigosTier))
}
