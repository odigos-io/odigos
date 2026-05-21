package opamp

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestResolveTransport(t *testing.T) {
	t.Parallel()

	require.Equal(t, OpAmpTransportHTTP,
		ResolveTransport("", true, common.K8sInitContainerMountMethod, ""))

	require.Equal(t, OpAmpTransportUnix,
		ResolveTransport(OpAmpTransportUnix, true, common.K8sVirtualDeviceMountMethod, "17"))
	require.Equal(t, OpAmpTransportNone,
		ResolveTransport(OpAmpTransportUnix, true, common.K8sVirtualDeviceMountMethod, "1.8.0"))
	require.Equal(t, OpAmpTransportNone,
		ResolveTransport(OpAmpTransportUnix, true, common.K8sInitContainerMountMethod, "21"))

	require.Equal(t, OpAmpTransportHTTP,
		ResolveTransport(OpAmpTransportHTTP, true, common.K8sInitContainerMountMethod, "8"))

	require.Equal(t, OpAmpTransportNone,
		ResolveTransport("", false, common.K8sVirtualDeviceMountMethod, ""))
}
